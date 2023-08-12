package lookup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/danthegoodman1/GoAPITemplate/utils"
	"io"
	"net/http"
	"strconv"
)

var (
	ErrNoPathPrefix = errors.New("no path prefix for virtual bucket")
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

	jBytes, err := json.Marshal(VirtualBucketResolveReq{
		VirtualBucket: virtBucket,
	})
	if err != nil {
		return nil, fmt.Errorf("error in json.Marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, utils.LookupURL+"/resolve_virtual_bucket", "POST", bytes.NewReader(jBytes))
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

	err = json.Unmarshal(resBytes, &resBody)
	if err != nil {
		return nil, fmt.Errorf("error in json.Unmarshal: %w", err)
	}

	if resBody.Prefix == "" {
		return nil, fmt.Errorf("virtual bucket '%s': %w", virtBucket, ErrNoPathPrefix)
	}

	return &resBody, nil
}
