package http_server

import (
	"encoding/xml"
	"testing"
)

func TestListObjectResult(t *testing.T) {
	res := ListBucketResult{
		XMLName:     xml.Name{},
		IsTruncated: false,
		Contents: []Content{
			{
				Key:          "some/sample.parquet",
				Size:         1024,
				StorageClass: "STANDARD",
			},
		},
		Name:                  "bucket name",
		MaxKeys:               1000,
		EncodingType:          "url",
		KeyCount:              500,
		ContinuationToken:     "blah",
		NextContinuationToken: "moreblah",
	}
	resb, err := xml.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(resb))
}
