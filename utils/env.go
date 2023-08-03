package utils

import "os"

var (
	Env = os.Getenv("ENV")

	CRDB_DSN   = os.Getenv("CRDB_DSN")
	S3ProxyUrl = os.Getenv("S3_PROXY_URL")
)
