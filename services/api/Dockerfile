# Dockerfile for the Civic Intelligence Go BFF (services/api).
#
# The Go module has many `replace` directives that point at sibling packages
# (../../adapters/kenya, ../../packages/auth, etc.), so the build context MUST
# be the repo root, NOT services/api/. Build from the repo root:
#
#   docker build -f services/api/Dockerfile -t civic-api .
#
# Or with the Railway / Render / Fly.io build pipeline, set the Dockerfile
# path to `services/api/Dockerfile` and the root directory to the repo root.
#
# The image runs the BFF on :9000. Override with `docker run -p 8080:9000
# -e API_SERVICE_ADDR=:9000 civic-api`.

# ---- Stage 1: build ----------------------------------------------------
FROM golang:1.23-alpine AS builder

# Install git (go mod needs it for some transitive deps) + ca-certificates
# (so HTTPS calls to kenyalaw.org / parliament.go.ke work in the final image).
RUN apk add --no-cache git ca-certificates linux-headers

WORKDIR /src

# Cache module downloads. Copy go.mod + go.sum first so a code change doesn't
# invalidate the module layer.
COPY services/api/go.mod services/api/go.sum ./services/api/
# Copy ALL go.mod / go.sum files referenced by replace directives so the
# workspace's full module graph is visible to `go mod download`.
COPY adapters/ ./adapters/
COPY packages/ ./packages/
COPY services/legislation/ ./services/legislation/
COPY services/simulation/ ./services/simulation/

# Download deps for the API module. The replace directives resolve to local
# paths, so all the sibling modules need to be present (above COPYs).
RUN cd services/api && go mod download

# Copy the rest of the API source. The sibling packages were already copied
# above for `go mod download`, but they may have changed — copy again so
# the build picks up the latest code.
COPY services/api/ ./services/api/

# Build a statically-linked binary so the final image can be `FROM scratch`
# or `alpine` without glibc concerns. CGO is disabled by default on alpine.
RUN cd services/api && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.version=docker" -o /out/civic-api ./cmd

# ---- Stage 2: runtime --------------------------------------------------
FROM alpine:3.20

# ca-certificates for HTTPS calls to kenyalaw.org / parliament.go.ke.
# tzdata so time.LoadLocation("Africa/Nairobi") works for the calendar.
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S civic && adduser -S civic -G civic

# Copy the binary from the builder stage.
COPY --from=builder /out/civic-api /usr/local/bin/civic-api

# Run as the non-root user created above.
USER civic

# The BFF listens on :9000 by default (see Config.HTTPAddr in cmd/main.go).
# Override with the API_SERVICE_ADDR env var.
ENV API_SERVICE_ADDR=:9000
EXPOSE 9000

# Healthcheck — the BFF exposes /api/v1/healthz (registered in cmd/main.go).
# Fails fast (2s timeout) so the orchestrator restarts the container quickly
# if the BFF hangs on startup.
HEALTHCHECK --interval=30s --timeout=2s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:9000/api/v1/healthz || exit 1

ENTRYPOINT ["civic-api"]
