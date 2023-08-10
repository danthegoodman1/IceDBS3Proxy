package utils

import (
	"os"
	"strings"
)

var (
	Env = os.Getenv("ENV")

	CRDB_DSN      = os.Getenv("CRDB_DSN")
	S3ProxyUrl    = os.Getenv("S3_PROXY_URL")
	S3ProxyUrlPtr = Ptr(S3ProxyUrl)
	MyURL         = os.Getenv("MY_URL")
	MyURLParts    = strings.Split(MyURL, ".")
	AWSKeyID      = os.Getenv("AWS_KEY_ID")
	AWSSecretKey  = os.Getenv("AWS_SECRET_KEY")
	S3Bucket      = os.Getenv("S3_BUCKET")
	S3BucketPtr   = Ptr(S3Bucket)
	S3UsePath     = os.Getenv("S3_USE_PATH") == "1"
	AWSRegion     = os.Getenv("AWS_REGION")
	AWSRegionPtr  = Ptr(AWSRegion)
)
