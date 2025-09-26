# --- build stage (Debian) ---
FROM golang:1.22-bookworm AS build
ENV GOTOOLCHAIN=auto
WORKDIR /src

# CGO toolchain + SQLite headers
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      build-essential pkg-config libsqlite3-dev ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Fail fast if sqlc output is missing
RUN test -e ./internal/db && test -n "$(ls -A ./internal/db)" || \
  (echo "ERROR: ./internal/db (sqlc output) missing. Run 'sqlc generate -f db/sqlc.yaml' and commit." && exit 1)

# Build natively for the image arch (Apple Silicon -> linux/arm64)
ARG TARGETOS
ARG TARGETARCH
ARG GO_BUILD_FLAGS
RUN CGO_ENABLED=1 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -v -x -o /out/server -trimpath $GO_BUILD_FLAGS ./cmd/server

# --- runtime stage (Debian slim) ---
FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates tzdata wget libsqlite3-0 && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /out/server /app/server
RUN mkdir -p /app/public /data/cache /data/exports /data/pmtiles
EXPOSE 8080
ENV PORT=8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=10 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/server"]
