package icedb

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/bytedance/sonic"
	"github.com/danthegoodman1/GoAPITemplate/utils"
	"github.com/rs/zerolog"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoLogFiles          = errors.New("no log files found in s3")
	ErrNoAliveFiles        = errors.New("no alive files")
	ErrColumnTypeCollision = errors.New("column type collision")
)

type (
	IceDBLogReader struct {
		s3Client *s3.Client
	}
)

func NewIceDBLogReader(ctx context.Context) (*IceDBLogReader, error) {
	logReader := &IceDBLogReader{}
	s3Creds := credentials.NewStaticCredentialsProvider(utils.AWSKeyID, utils.AWSSecretKey, "")
	s3Cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(s3Creds), config.WithRegion(utils.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("error in config.LoadDefaultConfig: %w", err)
	}
	logReader.s3Client = s3.NewFromConfig(s3Cfg, func(options *s3.Options) {
		options.BaseEndpoint = utils.S3UrlPtr
		options.UsePathStyle = utils.S3UsePath
	})
	return logReader, nil
}

type (
	LogSnapshot struct {
		AliveFiles []FileMarker
		Schema     Schema
	}

	LogMeta struct {
		Version             int  `json:"v"`
		TimestampMS         int  `json:"t"`
		SchemaStartLine     int  `json:"sch"`
		FileMarkerStartLine int  `json:"f,omitempty"`
		TombstoneStartLine  *int `json:"tmb,omitempty"`
	}

	Schema map[string]string

	Tombstone struct {
		Path        string `json:"p"`
		TimestampMS int    `json:"t"`
	}

	FileMarker struct {
		Path        string `json:"p"`
		ByteLength  int    `json:"b"`
		TimestampMS int    `json:"t"`
		Tombstone   *int   `json:"tmb,omitempty"`
	}
)

// Offset is with the path prefix
func (lr *IceDBLogReader) ReadState(ctx context.Context, pathPrefix, offset string, maxMS, maxItems int64) (*LogSnapshot, error) {
	if maxMS == 0 {
		maxMS = time.Now().UnixMilli()
	}
	logger := zerolog.Ctx(ctx)
	var contToken *string
	snapshot := LogSnapshot{
		AliveFiles: []FileMarker{},
		Schema:     make(map[string]string),
	}
	var s3Files []types.Object
	prefix := strings.Join([]string{pathPrefix, "_log"}, "/")
	for {
		logger.Debug().Str("prefix", prefix).Str("realBucket", *utils.S3BucketPtr).Msgf("listing s3 objects")

		listObjects, err := lr.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            utils.S3BucketPtr,
			ContinuationToken: contToken,
			MaxKeys:           1000,
			Prefix:            &prefix,
		})
		if err != nil {
			return nil, fmt.Errorf("error in : %w", err)
		}
		logger.Debug().Msgf("got %d items in list", len(listObjects.Contents))
		for _, object := range listObjects.Contents {
			key := *object.Key
			ts, _, err := getLogFileInfo(key)
			if err != nil {
				return nil, fmt.Errorf("error in getLogFileInfo for file %s: %w", *object.Key, err)
			}
			if ts <= maxMS {
				s3Files = append(s3Files, object)
			}
		}
		if !listObjects.IsTruncated {
			// Break if we are over or if we are at the end of the list
			break
		}
	}
	logger.Debug().Msgf("finished listing s3 objects with length %d", len(s3Files))
	if len(s3Files) == 0 {
		return nil, ErrNoLogFiles
	}

	// Ensure they are sorted
	slices.SortFunc(s3Files, func(a, b types.Object) int {
		if *a.Key > *b.Key {
			return 1
		} else if *a.Key < *b.Key {
			return -1
		}
		return 0
	})

	aliveFiles := map[string]FileMarker{}
	for _, object := range s3Files {
		obj, err := lr.s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: utils.S3BucketPtr,
			Key:    object.Key,
		})
		if err != nil {
			return nil, fmt.Errorf("error in GetObject for file %s: %w", *object.Key, err)
		}
		defer obj.Body.Close()
		fileBytes, err := io.ReadAll(obj.Body)
		if err != nil {
			return nil, fmt.Errorf("error in io.ReadAll for file %s: %w", *object.Key, err)
		}
		fileLines := strings.Split(string(fileBytes), "\n")
		var meta LogMeta
		err = sonic.Unmarshal([]byte(fileLines[0]), &meta)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling meta for file %s: %w", *object.Key, err)
		}

		var schema Schema
		err = sonic.Unmarshal([]byte(fileLines[meta.SchemaStartLine]), &schema)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling schema for file %s: %w", *object.Key, err)
		}

		// Aggregate the schema
		for colName, colType := range schema {
			if eType, exists := snapshot.Schema[colName]; exists && eType != colType {
				return nil, fmt.Errorf("col %s types %s %s: %w", colName, colType, eType, ErrColumnTypeCollision)
			} else {
				snapshot.Schema[colName] = colType
			}
		}

		// Determine alive files
		for i := meta.FileMarkerStartLine; i < len(fileLines); i++ {
			var fm FileMarker
			err = sonic.Unmarshal([]byte(fileLines[i]), &fm)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling file marker line %d for file %s: %w", i, *object.Key, err)
			}
			if _, exists := aliveFiles[fm.Path]; fm.Tombstone != nil && exists {
				// found a tombstone for the file, remove it
				delete(aliveFiles, fm.Path)
			} else if fm.Tombstone == nil {
				aliveFiles[fm.Path] = fm
			}
		}
	}

	if len(aliveFiles) == 0 {
		return nil, ErrNoAliveFiles
	}

	// Need the final list to do offset and limit, otherwise we
	for _, file := range aliveFiles {
		snapshot.AliveFiles = append(snapshot.AliveFiles, file)
	}

	// Sort
	slices.SortFunc(snapshot.AliveFiles, func(a, b FileMarker) int {
		if a.Path > b.Path {
			return 1
		}
		if a.Path < b.Path {
			return -1
		}
		return 0
	})

	// Check for offset
	if offset != "" {
		ind := slices.IndexFunc(snapshot.AliveFiles, func(marker FileMarker) bool {
			return marker.Path > offset
		})
		if ind != -1 && len(snapshot.AliveFiles) > ind {
			snapshot.AliveFiles = snapshot.AliveFiles[ind+1:]
		}
	}

	// Limit
	if maxItems != 0 && maxItems < int64(len(snapshot.AliveFiles)) {
		snapshot.AliveFiles = snapshot.AliveFiles[:maxItems]
	}

	return &snapshot, nil
}

func getLogFileInfo(fileName string) (int64, bool, error) {
	splitDir := strings.Split(fileName, "/")
	fileName = splitDir[len(splitDir)-1]
	nameParts := strings.Split(fileName, "_")
	fileTs, err := strconv.Atoi(nameParts[0])
	if err != nil {
		return 0, false, fmt.Errorf("error in Atoi: %w", err)
	}
	var merged bool
	if len(nameParts) > 2 && nameParts[1] == "m" {
		merged = true
	} else {
		merged = false
	}
	return int64(fileTs), merged, nil
}
