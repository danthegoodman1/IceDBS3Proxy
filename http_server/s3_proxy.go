package http_server

import (
	"encoding/xml"
	"fmt"
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
	domainParts := strings.Split(c.Request().Host, ".")
	isPathRouting := false
	if len(domainParts) > 2 {
		// vhost routing
		virtualBucketName = domainParts[0]
	} else {
		// path routing
		isPathRouting = true
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
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("virtualBucketName", virtualBucketName).Bool("isPathRouting", isPathRouting)
	})
	logger.Debug().Msg("got list request")

	// realBucketName := ""    // TODO: from lookup
	var contents []Content
	for i := 1; i <= 10000; i++ {
		contents = append(contents, Content{
			Key:          fmt.Sprintf("some-path/%d.parquet", i),
			Size:         1024,
			StorageClass: "STANDARD",
		})
	}

	res := ListBucketResult{
		XMLName:     xml.Name{},
		IsTruncated: false, // let's just serve all of them
		// Contents: []Content{
		// 	{
		// 		Key:          "some/sample.parquet",
		// 		Size:         1024,
		// 		StorageClass: "STANDARD",
		// 	},
		// 	{
		// 		Key:          "another/path/totally/sample.parquet",
		// 		Size:         2048,
		// 		StorageClass: "STANDARD",
		// 	},
		// },
		Contents:     contents,
		Name:         virtualBucketName,
		MaxKeys:      10000,
		EncodingType: "url",
		KeyCount:     10000,
	}

	// Look up files

	return c.XML(http.StatusOK, res)
}
