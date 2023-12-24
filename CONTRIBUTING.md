# Contributing

## How to run

Start local minio:

```
docker compose down -v && docker compose up -d
```

In another terminal, run clickhouse local:

```
clickhouse local
```

### Setup

Write some initial files:

I use the simple test with icedb and just comment out the delete at the end


Select the files directly from minio to make sure they work:

```
select count() from s3('http://localhost:9000/testbucket/example/_data/**/*.parquet', 'user', 'password')
```
```
select user_id, count() as events
from s3('http://localhost:9000/testbucket/example/_data/**/*.parquet', 'user', 'password')
group by user_id
order by events
```

Run IceDBS3Proxy:
```
task
```

And the example control plane:
```
bun --watch run example_control_plane/index.ts
```

Then, make the bucket public access (needed to be able to proxy requests)

Request to a fake bucket:

```
select count() from s3('http://localhost:8080/fakebucket/**/*.parquet', 'iceuser', 'icepassword')
```

There is a session S3 object cache, so sometimes it's easier to just use:

```
clickhouse local -q "select count() from s3('http://localhost:8080/fakebucket/**/*.parquet', 'iceuser', 'icepassword')"
```

```
select user_id, count() as events
from s3('http://localhost:8080/fakebucket/**/*.parquet', 'iceuser', 'icepassword')
group by user_id
order by events
```