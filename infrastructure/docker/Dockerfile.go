# Multi-stage Go build for any service in services/*
# Usage: docker build -f infrastructure/docker/Dockerfile.go -t civic/api --build-arg SERVICE=api ../..

FROM golang:1.22-alpine AS builder
WORKDIR /src
ARG SERVICE
COPY go.mod go.sum ./
COPY packages ./packages
COPY services/${SERVICE} ./services/${SERVICE}
COPY adapters ./adapters
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service ./services/${SERVICE}/cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/service /app/service
USER nonroot
EXPOSE 9000
ENTRYPOINT ["/app/service"]
