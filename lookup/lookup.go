package lookup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/danthegoodman1/GoAPITemplate/utils"
	"github.com/mailgun/groupcache/v2"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoPathPrefix = errors.New("no path prefix for virtual bucket")
	poolServer      *http.Server
	group           *groupcache.Group
)

type (
	VirtualBucketResolveReq struct {
		VirtualBucket string
	}
	VirtualBucketResolveRes struct {
		// If omitted, will be ""
		Prefix string
		// If omitted, will be current time
		TimeMS *int64
	}
)

func InitCache(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	pool := groupcache.NewHTTPPoolOpts(utils.CacheSelfAddr, &groupcache.HTTPPoolOptions{})

	// Add more peers to the cluster You MUST Ensure our instance is included in this list else
	// determining who owns the key across the cluster will not be consistent, and the pool won't
	// be able to determine if our instance owns the key.
	pool.Set(utils.CachePeers...)

	poolServer = &http.Server{
		Addr:    strings.Split(utils.CacheSelfAddr, "://")[1],
		Handler: pool,
	}

	// Start an HTTP server to listen for peer requests from the groupcache
	go func() {
		logger.Debug().Msg("cache pool server listening...")
		if err := poolServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("error on pool server listen")
		}
	}()

	group = groupcache.NewGroup("virtual_buckets", 3000000, groupcache.GetterFunc(
		func(ctx context.Context, virtualBucket string, dest groupcache.Sink) error {
			jBytes, err := sonic.Marshal(VirtualBucketResolveReq{
				VirtualBucket: virtualBucket,
			})
			if err != nil {
				return fmt.Errorf("error in sonic.Marshal: %w", err)
			}

			res, err := resolveFromAPI(ctx, jBytes)
			if err != nil {
				return fmt.Errorf("error in resolveFromAPI: %w", err)
			}

			return dest.SetBytes(res, time.Now().Add(time.Second*time.Duration(utils.CacheTTLSeconds)))
		},
	))
}

func CloseCache(ctx context.Context) error {
	return poolServer.Shutdown(ctx)
}

func resolveFromAPI(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, utils.LookupURL+"/resolve_virtual_bucket", "POST", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error in NewRequestWithContext: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error in http.Do: %w", err)
	}

	defer res.Body.Close()

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error in io.ReadAll: %w", err)
	}
	return resBytes, nil
}

// ResolveVirtualBucket Lookups up a prefix (namespace) and timestamp for a given virtual bucket.
// The remote server must return a prefix. If a timestamp is not returned then the current one will be used.
func ResolveVirtualBucket(ctx context.Context, virtBucket string) (*VirtualBucketResolveRes, error) {
	var resBody VirtualBucketResolveRes
	if utils.DevLookupPrefix != "" {
		resBody.Prefix = utils.DevLookupPrefix
		if utils.DevLookupTimeMS != "" {
			timeMS, err := strconv.Atoi(utils.DevLookupTimeMS)
			if err != nil {
				return nil, fmt.Errorf("error in Atoi(DevLookupTimeMS): %w", err)
			}
			resBody.TimeMS = utils.Ptr(int64(timeMS))
		}
		return &resBody, nil
	}

	var jBytes []byte
	var err error
	if utils.CacheEnabled {
		if err := group.Get(ctx, virtBucket, groupcache.AllocatingByteSliceSink(&jBytes)); err != nil {
			return nil, fmt.Errorf("error getting from groupcache: %w", err)
		}
	} else {
		jBytes, err = sonic.Marshal(VirtualBucketResolveReq{
			VirtualBucket: virtBucket,
		})
		if err != nil {
			return nil, fmt.Errorf("error in sonic.Marshal: %w", err)
		}

		resBytes, err := resolveFromAPI(ctx, jBytes)
		if err != nil {
			return nil, fmt.Errorf("error in resolveFromAPI: %w", err)
		}

		err = sonic.Unmarshal(resBytes, &resBody)
		if err != nil {
			return nil, fmt.Errorf("error in sonic.Unmarshal: %w", err)
		}
	}

	if resBody.Prefix == "" {
		return nil, fmt.Errorf("virtual bucket '%s': %w", virtBucket, ErrNoPathPrefix)
	}

	return &resBody, nil
}
