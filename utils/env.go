package utils

import (
	"os"
	"strings"
)

var (
	Env = os.Getenv("ENV")

	CRDB_DSN   = os.Getenv("CRDB_DSN")
	S3ProxyUrl = os.Getenv("S3_PROXY_URL")
	MyURL      = os.Getenv("MY_URL")
	MyURLParts = strings.Split(MyURL, ".")
)
