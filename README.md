# IceDB S3 Proxy

The IceDB S3 Proxy sits between S3 and your query engine (ClickHouse, DuckDB, CHDB, datafusion, pandas, etc.) and spoofs a bucket that only has the alive data files in it.

The proxy takes in a virtual bucket, requests to an API to resolve that to a real bucket and some prefix (e.g. namespace/tenant), and spoofs List, Head, and Get requests.

For List requests, only the IceDB log is read and returned. For Head and Get requests, the request to S3 is intercepted and the virtual bucket is swapped for the real bucket + prefix, and the auth header is ripped off.

Currently, the proxy expects the target bucket to require no auth, as found in the case of local minio or an AWS VPC endpoint. It only requires read access to buckets.

<!-- TOC -->
* [IceDB S3 Proxy](#icedb-s3-proxy)
  * [Lookup](#lookup)
  * [Configuration](#configuration)
  * [Control Plane](#control-plane)
  * [Performance](#performance)
<!-- TOC -->

## Lookup

You need to lookup virtual buckets to real buckets.

You can put a path prefix in the `LOOKUP_URL`. `LOOKUP_AUTH` will be passed in (blank string if not provided)

## Configuration

Check [the environment file for parameters](utils/env.go) :)

## Control Plane

If `TimeMS == 0`, then the current time of the operation will be used. This no longer guarantees stable snapshots, but is otherwise safe. This is included in cache so queries against a cached lookup will still use current time.

## Performance

Faster than querying S3 directly with fully merge icedb table, and that benefit grows as the number of data files grows.

Test (cold runs):

```
SELECT count()
FROM s3('https://s3.us-east-1.amazonaws.com/icedb-test-tangia-staging/chicago_taxis_1m/_data/**/*.parquet')

Query id: 81e70392-1739-41c9-a372-1822bd6f5596

┌───count()─┐
│ 209512921 │
└───────────┘

1 row in set. Elapsed: 1.187 sec. Processed 209.51 million rows, 3.54 KB (176.53 million rows/s., 2.98 KB/s.)
Peak memory usage: 137.54 MiB.

ip-10-0-166-70.ec2.internal :) select count() from s3('http://localhost:8080/fakebucket/**/*.parquet')

SELECT count()
FROM s3('http://localhost:8080/fakebucket/**/*.parquet')

Query id: e02fa4f7-b363-49e1-87c7-14bc0953b9c0

┌───count()─┐
│ 209512921 │
└───────────┘

1 row in set. Elapsed: 1.114 sec. Processed 209.51 million rows, 3.54 KB (188.07 million rows/s., 3.17 KB/s.)
Peak memory usage: 138.45 MiB.
```