package utils

import (
	"os"
	"strings"
)

var (
	Env = os.Getenv("ENV")

	S3ProxyUrl    = MustEnv("S3_PROXY_URL")
	S3ProxyUrlPtr = Ptr(S3ProxyUrl)

	MyHost     = MustEnv("MY_HOST")
	MyURLParts = strings.Split(MyHost, ".")

	AWSKeyID     = MustEnv("AWS_KEY_ID")
	AWSSecretKey = MustEnv("AWS_SECRET_KEY")
	S3Bucket     = MustEnv("S3_BUCKET")
	S3BucketPtr  = Ptr(S3Bucket)
	S3UsePath    = os.Getenv("S3_USE_PATH") == "1"
	AWSRegion    = MustEnv("AWS_REGION")
	AWSRegionPtr = Ptr(AWSRegion)

	LookupURL  = MustEnv("LOOKUP_URL")
	LookupAuth = os.Getenv("LOOKUP_AUTH")

	CacheEnabled = os.Getenv("CACHE_ENABLED") == "1"
	// http://x:y,http://z:y,... MUST INCLUDE SELF! Only need to include self to cache as a single node
	CachePeers = strings.Split(os.Getenv("CACHE_PEERS"), ",")
	// http://x.x.x.x:yyyy
	CacheSelfAddr   = os.Getenv("CACHE_SELF_ADDR")
	CacheBytes      = GetEnvOrDefaultInt("CACHE_BYTES", 100_000_000) // 100MB
	CacheTTLSeconds = GetEnvOrDefaultInt("CACHE_SECONDS", 10)

	DevLookupPrefix = os.Getenv("DEV_LOOKUP_PREFIX")
	DevLookupTimeMS = os.Getenv("DEV_LOOKUP_TIME_MS")
)
