FROM golang:1.21.0-bullseye as build

WORKDIR /app

COPY go.* /app/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=ssh \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build $GO_ARGS -o /app/icedbs3proxy

# Need glibc
FROM gcr.io/distroless/base-debian11

ENTRYPOINT ["/app/icedbs3proxy"]
COPY --from=build /app/icedbs3proxy /app/
