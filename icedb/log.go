package icedb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/danthegoodman1/GoAPITemplate/utils"
	"github.com/rs/zerolog"
	"io"
	"slices"
	"strconv"
	"strings"
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
		options.BaseEndpoint = utils.S3ProxyUrlPtr
		options.UsePathStyle = utils.S3UsePath
	})
	return logReader, nil
}

type (
	LogSnapshot struct {
		AliveFiles map[string]FileMarker
		Schema     Schema
	}

	LogMeta struct {
		Version             int  `json:"v"`
		TimestampMS         int  `json:"t"`
		SchemaStartLine     int  `json:"sch"`
		FileMarkerStartLine int  `json:"f"`
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

func (lr *IceDBLogReader) ReadState(ctx context.Context, pathprefix string, maxMS int64) (*LogSnapshot, error) {
	logger := zerolog.Ctx(ctx)
	var contToken *string
	snapshot := LogSnapshot{
		AliveFiles: make(map[string]FileMarker),
		Schema:     make(map[string]string),
	}
	var s3Files []types.Object
	for {
		logger.Debug().Msgf("listing s3 objects")
		listObjects, err := lr.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            utils.S3BucketPtr,
			ContinuationToken: contToken,
			MaxKeys:           1000,
			Prefix:            utils.Ptr(strings.Join([]string{pathprefix, "_log"}, "/")),
		})
		if err != nil {
			return nil, fmt.Errorf("error in : %w", err)
		}
		for _, object := range listObjects.Contents {
			ts, _, err := getLogFileInfo(*object.Key)
			if err != nil {
				return nil, fmt.Errorf("error in getLogFileInfo for file %s: %w", *object.Key, err)
			}
			if ts <= maxMS {
				s3Files = append(s3Files, object)
			}
		}
		if !listObjects.IsTruncated {
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
		err = json.Unmarshal([]byte(fileLines[0]), &meta)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling meta for file %s: %w", *object.Key, err)
		}

		var schema Schema
		err = json.Unmarshal([]byte(fileLines[meta.SchemaStartLine]), &schema)
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
			err = json.Unmarshal([]byte(fileLines[i]), &fm)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling file marker line %d for file %s: %w", i, *object.Key, err)
			}
			if _, exists := snapshot.AliveFiles[fm.Path]; fm.Tombstone != nil && exists {
				// found a tombstone for the file, remove it
				delete(snapshot.AliveFiles, fm.Path)
			} else {
				// otherwise add to alive files
				snapshot.AliveFiles[fm.Path] = fm
			}
		}
	}

	if len(snapshot.AliveFiles) == 0 {
		return nil, ErrNoAliveFiles
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
