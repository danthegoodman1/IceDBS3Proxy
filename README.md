# IceDB S3 Proxy

The IceDB S3 Proxy sits between S3 and your query engine (ClickHouse, DuckDB, CHDB, datafusion, pandas, etc.) and spoofs a bucket that only has the alive data files in it.

The proxy takes in a virtual bucket, requests to an API to resolve that to a real bucket and some prefix (e.g. namespace/tenant), and spoofs List, Head, and Get requests.

For List requests, only the IceDB log is read and returned. For Head and Get requests, the request to S3 is intercepted and the virtual bucket is swapped for the real bucket + prefix, and the auth header is ripped off.

Currently, the proxy expects the target bucket to require no auth, as found in the case of local minio or an AWS VPC endpoint. It only requires read access to buckets.

<!-- TOC -->
* [IceDB S3 Proxy](#icedb-s3-proxy)
  * [Lookup](#lookup)
<!-- TOC -->

## Lookup

You need to lookup virtual buckets to real buckets.

You can put a path prefix in the `LOOKUP_URL`. `LOOKUP_AUTH` will be passed in (blank string if not provided)

## Configuration

Check [the environment file for parameters](utils/env.go) :)