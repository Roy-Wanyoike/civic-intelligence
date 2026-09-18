//
// Package storage is a thin wrapper around github.com/minio/minio-go/v7.
//
// It works with **both AWS S3 and MinIO** — the only difference is the
// endpoint URL (and TLS). MinIO is fully S3-compatible, so a single client
// implementation covers both:
//
//   * AWS S3      → endpoint "s3.amazonaws.com", TLS on
//   * MinIO local → endpoint "127.0.0.1:9000",  TLS off
//   * MinIO prod  → endpoint "minio.internal",   TLS on
//
// Methods:
//   - Upload(ctx, bucket, key, reader) → stores an object
//   - Download(ctx, bucket, key)       → returns io.ReadCloser
//   - Delete(ctx, bucket, key)         → removes an object
//   - Exists(ctx, bucket, key)         → bool probe (StatObject)
//   - Presign(ctx, bucket, key, expiry)→ time-limited URL
//
// The wrapper exposes a small ObjectStorage interface; consumers depend on
// the interface, not the concrete *minio.Client, so tests can substitute a
// mock (see s3_test.go).
package storage

import (
        "context"
        "errors"
        "fmt"
        "io"
        "net/url"
        "strings"
        "time"

        "github.com/minio/minio-go/v7"
        "github.com/minio/minio-go/v7/pkg/credentials"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config describes how to connect to the object storage backend.
type Config struct {
        // Endpoint is the host[:port] of the S3/MinIO server. Empty defaults to
        // AWS S3 ("s3.amazonaws.com").
        Endpoint string

        // Region for AWS S3 (ignored for MinIO unless V4 signing requires it).
        Region string

        // AccessKey / SecretKey for static credentials. Leave empty to use the
        // IAM role on the EC2/EKS node (credentials.Static + env chain).
        AccessKey string
        SecretKey string

        // UseTLS enables HTTPS. Set false only for local MinIO dev.
        UseTLS bool

        // DefaultBucket is used by callers that don't supply their own.
        DefaultBucket string

        // PresignExpiry is the default expiry for presigned URLs.
        PresignExpiry time.Duration
}

// DefaultConfig returns a config that works against a local MinIO dev
// container (the docker-compose ships one).
func DefaultConfig() Config {
        return Config{
                Endpoint:      "127.0.0.1:9000",
                Region:        "us-east-1",
                AccessKey:     "minioadmin",
                SecretKey:     "minioadmin",
                UseTLS:        false,
                DefaultBucket: "civic-intelligence",
                PresignExpiry: 15 * time.Minute,
        }
}

// ---------------------------------------------------------------------------
// ObjectStorage interface + concrete implementation
// ---------------------------------------------------------------------------

// ObjectStorage is the interface every consumer depends on.
type ObjectStorage interface {
        // Upload stores the contents of reader under bucket/key. contentType
        // is the MIME type; size is the byte length (or -1 for unknown).
        Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error

        // Download returns a ReadCloser for the object. The caller MUST Close it.
        Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)

        // Delete removes one object.
        Delete(ctx context.Context, bucket, key string) error

        // Exists returns true if the object exists.
        Exists(ctx context.Context, bucket, key string) (bool, error)

        // Presign returns a time-limited URL that grants the bearer GET access
        // to the object. Used for direct-to-browser downloads.
        Presign(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)

        // Ping checks the backend is reachable (ListBuckets with 1 result).
        Ping(ctx context.Context) error

        // Close releases any pooled connections (no-op for minio-go, which
        // uses http.DefaultClient underneath).
        Close() error
}

// Compile-time assertion that *s3Client implements ObjectStorage.
var _ ObjectStorage = (*s3Client)(nil)

// s3Client wraps *minio.Client.
type s3Client struct {
        cfg    Config
        client MinioAPI
}

// MinioAPI is the subset of *minio.Client we use. Declaring it as an
// interface lets us drop a mock into s3Client.client during tests.
type MinioAPI interface {
        PutObject(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
        GetObject(ctx context.Context, bucket, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
        RemoveObject(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error
        StatObject(ctx context.Context, bucket, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
        PresignedGetObject(ctx context.Context, bucket, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
        ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
}

// New constructs an ObjectStorage from Config.
func New(cfg Config) (ObjectStorage, error) {
        if cfg.Endpoint == "" {
                cfg.Endpoint = "s3.amazonaws.com"
        }
        cli, err := minio.New(cfg.Endpoint, &minio.Options{
                Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
                Secure: cfg.UseTLS,
                Region: cfg.Region,
        })
        if err != nil {
                return nil, fmt.Errorf("storage: minio client: %w", err)
        }
        return &s3Client{cfg: cfg, client: cli}, nil
}

// NewWithClient lets tests inject a mock MinioAPI. Production code should
// use New().
func NewWithClient(cfg Config, c MinioAPI) ObjectStorage {
        return &s3Client{cfg: cfg, client: c}
}

// ---------------------------------------------------------------------------
// Upload / Download / Delete / Exists / Presign / Ping
// ---------------------------------------------------------------------------

// Upload implements ObjectStorage.
func (s *s3Client) Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
        if err := validateBucketKey(bucket, key); err != nil {
                return err
        }
        if contentType == "" {
                contentType = "application/octet-stream"
        }
        if size < 0 {
                size = -1 // minio-go streams when size is -1
        }
        _, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
                ContentType: contentType,
        })
        if err != nil {
                return fmt.Errorf("storage: upload %s/%s: %w", bucket, key, err)
        }
        return nil
}

// Download implements ObjectStorage.
func (s *s3Client) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
        if err := validateBucketKey(bucket, key); err != nil {
                return nil, err
        }
        obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
        if err != nil {
                // Translate NoSuchKey → ErrNotFound so callers can errors.Is(err, ErrNotFound).
                return nil, wrapMinioErr(fmt.Errorf("storage: download %s/%s: %w", bucket, key, err))
        }
        // Stat to surface "not found" eagerly (otherwise the first Read fails).
        if _, err := obj.Stat(); err != nil {
                _ = obj.Close()
                return nil, wrapMinioErr(err)
        }
        return obj, nil
}

// Delete implements ObjectStorage.
func (s *s3Client) Delete(ctx context.Context, bucket, key string) error {
        if err := validateBucketKey(bucket, key); err != nil {
                return err
        }
        if err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
                return fmt.Errorf("storage: delete %s/%s: %w", bucket, key, err)
        }
        return nil
}

// Exists implements ObjectStorage. It uses StatObject; the error response
// from minio-go carries the S3 error code which we inspect to distinguish
// "missing" from a real failure.
func (s *s3Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
        if err := validateBucketKey(bucket, key); err != nil {
                return false, err
        }
        info, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
        if err != nil {
                if isNotFound(err) {
                        return false, nil
                }
                return false, fmt.Errorf("storage: stat %s/%s: %w", bucket, key, err)
        }
        // A zero-size tombstone is theoretically possible; treat any successful
        // stat as "exists".
        _ = info
        return true, nil
}

// Presign implements ObjectStorage.
func (s *s3Client) Presign(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
        if err := validateBucketKey(bucket, key); err != nil {
                return "", err
        }
        if expiry <= 0 {
                expiry = s.cfg.PresignExpiry
                if expiry <= 0 {
                        expiry = 15 * time.Minute
                }
        }
        u, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
        if err != nil {
                return "", fmt.Errorf("storage: presign %s/%s: %w", bucket, key, err)
        }
        return u.String(), nil
}

// Ping implements ObjectStorage.
func (s *s3Client) Ping(ctx context.Context) error {
        if _, err := s.client.ListBuckets(ctx); err != nil {
                return fmt.Errorf("storage: ping: %w", err)
        }
        return nil
}

// Close implements ObjectStorage (no-op — minio-go uses http.DefaultClient).
func (s *s3Client) Close() error { return nil }

// ---------------------------------------------------------------------------
// Helpers + errors
// ---------------------------------------------------------------------------

// ErrEmptyBucket / ErrEmptyKey are returned by validateBucketKey.
var (
        ErrEmptyBucket = errors.New("storage: empty bucket")
        ErrEmptyKey     = errors.New("storage: empty key")
)

func validateBucketKey(bucket, key string) error {
        if strings.TrimSpace(bucket) == "" {
                return ErrEmptyBucket
        }
        if strings.TrimSpace(key) == "" {
                return ErrEmptyKey
        }
        return nil
}

// isNotFound reports whether err is an S3 NoSuchKey error. minio-go
// surfaces this as *minio.ErrorResponse with Code == "NoSuchKey".
func isNotFound(err error) bool {
        if err == nil {
                return false
        }
        var er minio.ErrorResponse
        if errors.As(err, &er) {
                return er.Code == "NoSuchKey" || er.Code == "404"
        }
        // Fall back to substring match for opaque transports.
        return strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "404 Not Found")
}

// wrapMinioErr translates "not found" into a typed sentinel so callers can
// use errors.Is without depending on minio-go types.
func wrapMinioErr(err error) error {
        if err == nil {
                return nil
        }
        if isNotFound(err) {
                return ErrNotFound
        }
        return err
}

// ErrNotFound is returned by Download for missing objects.
var ErrNotFound = errors.New("storage: object not found")
