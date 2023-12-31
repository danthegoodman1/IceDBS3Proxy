package http_server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/danthegoodman1/GoAPITemplate/icedb"
	"github.com/danthegoodman1/GoAPITemplate/lookup"
	"github.com/danthegoodman1/GoAPITemplate/utils"
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ListBucketResult struct {
	XMLName               xml.Name       `xml:"ListBucketResult,omitempty"`
	IsTruncated           bool           `xml:"IsTruncated,omitempty"`
	Contents              []Content      `xml:"Contents,omitempty"`
	Name                  string         `xml:"Name,omitempty"`
	Prefix                string         `xml:"Prefix,omitempty"`
	Delimiter             string         `xml:"Delimiter,omitempty"`
	MaxKeys               int            `xml:"MaxKeys,omitempty"`
	CommonPrefixes        []CommonPrefix `xml:"CommonPrefixes,omitempty"`
	EncodingType          string         `xml:"EncodingType,omitempty"`
	KeyCount              int            `xml:"KeyCount,omitempty"`
	ContinuationToken     string         `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string         `xml:"NextContinuationToken,omitempty"`
	StartAfter            string         `xml:"StartAfter,omitempty"`
}

type Content struct {
	ChecksumAlgorithm string        `xml:"ChecksumAlgorithm,omitempty"`
	ETag              string        `xml:"ETag,omitempty"`
	Key               string        `xml:"Key,omitempty"`
	LastModified      time.Time     `xml:"LastModified,omitempty"`
	Owner             Owner         `xml:"Owner,omitempty"`
	RestoreStatus     RestoreStatus `xml:"RestoreStatus,omitempty"`
	Size              int           `xml:"Size,omitempty"`
	StorageClass      string        `xml:"StorageClass,omitempty"`
}

type Owner struct {
	DisplayName string `xml:"DisplayName,omitempty"`
	ID          string `xml:"ID,omitempty"`
}

type RestoreStatus struct {
	IsRestoreInProgress bool      `xml:"IsRestoreInProgress,omitempty"`
	RestoreExpiryDate   time.Time `xml:"RestoreExpiryDate,omitempty"`
}

type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

type ListObjectRequest struct {
	ListType                 *int    `query:"list-type"`
	ContinuationToken        *string `query:"continuation-token"`
	Delimiter                *string `query:"delimiter"`
	EncodingType             *string `query:"encoding-type"`
	FetchOwner               *bool   `query:"fetch-owner"`
	MaxKeys                  *int    `query:"max-keys"`
	Prefix                   *string `query:"prefix"`
	StartAfter               *string `query:"start-after"`
	ExpectedBucketOwner      *string `header:"x-amz-expected-bucket-owner"`
	OptionalObjectAttributes *string `header:"x-amz-optional-object-attributes"`
	RequestPayer             *string `header:"x-amz-request-payer"`
}

func (srv *HTTPServer) ListObjectInterceptor(c *CustomContext) error {
	var req ListObjectRequest
	if err := c.Bind(&req); err != nil {
		return c.InternalError(err, "error binding")
	}

	var virtualBucketName string
	if c.IsPathRouting {
		// vhost routing
		domainParts := strings.Split(c.Request().Host, ".")
		virtualBucketName = domainParts[0]
	} else {
		// path routing
		u, err := url.Parse(c.Request().RequestURI)
		if err != nil {
			return c.InternalError(err, "error in url.Parse")
		}
		pathParts := strings.Split(u.Path, "/")
		if len(pathParts) == 0 {
			return c.String(http.StatusNotFound, "not found (invalid path")
		}
		virtualBucketName = pathParts[1]
	}

	logger := zerolog.Ctx(c.Request().Context())
	logger.Debug().Msg("got list request")

	maxKeys := utils.Deref(req.MaxKeys, 1000)

	// realBucketName := ""    // TODO: from lookup
	logReader, err := icedb.NewIceDBLogReader(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error in NewIceDBLogReader: %w", err)
	}

	// Resolve virtual bucket
	resolvedBucket, err := lookup.ResolveVirtualBucket(c.Request().Context(), c.VirtualBucketName, c.AWSCredentials.KeyID)
	if err != nil {
		return c.InternalError(err, "error in lookup.ResolveVirtualBucket")
	}

	// Prioritize ContinuationToken which is used if paginating (we force it to be last item), otherwise use StartAfter
	offset := utils.Deref(req.ContinuationToken, utils.Deref(req.StartAfter, ""))

	res := ListBucketResult{
		XMLName:      xml.Name{},
		IsTruncated:  false, // let's just serve all of them
		Name:         virtualBucketName,
		MaxKeys:      maxKeys,
		EncodingType: "url",
		KeyCount:     maxKeys,
	}

	aliveFiles, err := logReader.ReadState(c.Request().Context(), resolvedBucket.Prefix, offset, utils.Deref(resolvedBucket.TimeMS, time.Now().UnixMilli()), int64(utils.Deref(req.MaxKeys, 1000)))
	if errors.Is(err, icedb.ErrNoLogFiles) {
		// Just return no items
		return c.XML(http.StatusOK, res)
	}
	if err != nil {
		return c.InternalError(err, "error in ReadState")
	}

	var contents []Content
	for _, af := range aliveFiles.AliveFiles {
		contents = append(contents, Content{
			Key:          strings.ReplaceAll(af.Path, resolvedBucket.Prefix+"/_data/", ""), // drop the prefix
			Size:         af.ByteLength,
			StorageClass: "STANDARD",
		})
	}

	res.Contents = contents

	return c.XML(http.StatusOK, res)
}

func (srv *HTTPServer) CheckListOrGetObject(c *CustomContext) error {
	logger := zerolog.Ctx(c.Request().Context())

	// Can check this immediately, should catch ClickHouse and DuckDB
	if listType := c.QueryParam("list-type"); listType == "2" {
		logger.Debug().Msg("got list type query param, intercepting list request")
		return srv.ListObjectInterceptor(c)
	}

	if c.IsPathRouting {
		// path routing, path style list possibly
		u, err := url.Parse(c.Request().RequestURI)
		if err != nil {
			return c.InternalError(err, "error in url.Parse")
		}
		pathParts := strings.Split(u.Path, "/")

		if len(pathParts) == 3 && pathParts[2] == "" {
			// This is a `/bucket/` request, ListObject(V2)
			logger.Debug().Msg("request is list")
			return srv.ListObjectInterceptor(c)
		}
		// Otherwise we are `/bucket/**`, get object
		logger.Debug().Msg("request is get")
		return srv.ProxyS3Request(c)
	} else {
		// vhost routing
		return srv.ListObjectInterceptor(c)
	}
}

func (srv *HTTPServer) ProxyS3Request(c *CustomContext) error {
	logger := zerolog.Ctx(c.Request().Context())

	// Resolve virtual bucket
	resolvedBucket, err := lookup.ResolveVirtualBucket(c.Request().Context(), c.VirtualBucketName, c.AWSCredentials.KeyID)
	if err != nil {
		return c.InternalError(err, "error in lookup.ResolveVirtualBucket")
	}

	droppedVirtualBucket := strings.ReplaceAll(c.Request().RequestURI, c.VirtualBucketName, "")
	prefixWithData := resolvedBucket.Prefix + "/_data"

	newPath := prefixWithData + droppedVirtualBucket
	if c.IsPathRouting {
		// Need to deal with the bucket `/bucket/...` needs to be `/bucket/prefix/...
		// If the client is using path routing, we need to use path routing
		pathParts := strings.Split(droppedVirtualBucket, "/")
		newPathParts := []string{prefixWithData}
		if utils.S3UsePath {
			// if we are path routing, then we need to prepend the bucket
			newPathParts = []string{utils.S3Bucket, prefixWithData}
		}
		newPathParts = append(newPathParts, pathParts[2:]...)
		newPath = "/" + strings.Join(newPathParts, "/")
	}

	finalURL := utils.S3UrlWithSubdomain + newPath

	logger.UpdateContext(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Bool("proxied", true).Str("finalURL", finalURL)
	})
	logger.Debug().Str("authHeader", c.Request().Header.Get("Authorization")).Msg("proxying request")

	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method, finalURL, nil)
	if err != nil {
		return c.InternalError(err, "error making new request for proxying")
	}

	// Copy headers
	headers := c.Request().Header.Clone()
	// If we have an access key, throw it away, as it's partially based on the host
	headers.Del("Authorization")
	req.Header = headers

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return c.InternalError(err, "error doing proxy request")
	}
	defer res.Body.Close()

	// Copy headers
	for name, headers := range res.Header {
		// Iterate all headers with one name (e.g. Content-Type)
		for _, hdr := range headers {
			c.Response().Header().Set(name, hdr)
		}
	}

	return c.Stream(res.StatusCode, res.Header.Get("content-type"), res.Body)
}
