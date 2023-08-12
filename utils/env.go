package utils

import (
	"os"
	"strings"
)

var (
	Env = os.Getenv("ENV")

	S3ProxyUrl    = MustEnv("S3_PROXY_URL")
	S3ProxyUrlPtr = Ptr(S3ProxyUrl)
	MyURL         = MustEnv("MY_URL")
	MyURLParts    = strings.Split(MyURL, ".")
	AWSKeyID      = MustEnv("AWS_KEY_ID")
	AWSSecretKey  = MustEnv("AWS_SECRET_KEY")
	S3Bucket      = MustEnv("S3_BUCKET")
	S3BucketPtr   = Ptr(S3Bucket)
	S3UsePath     = os.Getenv("S3_USE_PATH") == "1"
	AWSRegion     = MustEnv("AWS_REGION")
	AWSRegionPtr  = Ptr(AWSRegion)

	LookupURL  = MustEnv("LOOKUP_URL")
	LookupAuth = os.Getenv("LOOKUP_AUTH")

	DevLookupPrefix = os.Getenv("DEV_LOOKUP_PREFIX")
	DevLookupTimeMS = os.Getenv("DEV_LOOKUP_TIME_MS")
)
