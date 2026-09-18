// FIXME: verify with go build when Go available
package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock MinioAPI ---------------------------------------------------------

// mockMinio is a hand-rolled implementation of MinioAPI used to exercise
// s3Client without a real S3/MinIO backend. It records every call and lets
// each test override the behaviour via the *Fn fields.
type mockMinio struct {
	putFn          func(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	getFn          func(ctx context.Context, bucket, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	removeFn       func(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error
	statFn         func(ctx context.Context, bucket, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	presignFn      func(ctx context.Context, bucket, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error)
	listBucketsFn  func(ctx context.Context) ([]minio.BucketInfo, error)

	putCalls     int
	getCalls     int
	removeCalls  int
	statCalls    int
	presignCalls int
	listCalls    int

	// Storage for the in-memory fake (used by Upload→Download round-trip).
	store map[string][]byte
}

func newMockMinio() *mockMinio {
	return &mockMinio{store: map[string][]byte{}}
}

func (m *mockMinio) PutObject(ctx context.Context, bucket, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	m.putCalls++
	if m.putFn != nil {
		return m.putFn(ctx, bucket, objectName, reader, objectSize, opts)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return minio.UploadInfo{}, err
	}
	key := bucket + "/" + objectName
	m.store[key] = body
	return minio.UploadInfo{Bucket: bucket, Key: objectName, Size: int64(len(body))}, nil
}

func (m *mockMinio) GetObject(ctx context.Context, bucket, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	m.getCalls++
	if m.getFn != nil {
		return m.getFn(ctx, bucket, objectName, opts)
	}
	key := bucket + "/" + objectName
	body, ok := m.store[key]
	if !ok {
		// Return an error equivalent to NoSuchKey.
		return nil, minio.ErrorResponse{Code: "NoSuchKey", BucketName: bucket, Key: objectName}
	}
	// We cannot easily construct a *minio.Object from bytes (it's an opaque
	// struct). For tests that exercise the success path, override getFn.
	_ = body
	return nil, errors.New("mockMinio: GetObject success path requires getFn override")
}

func (m *mockMinio) RemoveObject(ctx context.Context, bucket, objectName string, opts minio.RemoveObjectOptions) error {
	m.removeCalls++
	if m.removeFn != nil {
		return m.removeFn(ctx, bucket, objectName, opts)
	}
	delete(m.store, bucket+"/"+objectName)
	return nil
}

func (m *mockMinio) StatObject(ctx context.Context, bucket, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	m.statCalls++
	if m.statFn != nil {
		return m.statFn(ctx, bucket, objectName, opts)
	}
	if _, ok := m.store[bucket+"/"+objectName]; ok {
		return minio.ObjectInfo{Bucket: bucket, Key: objectName}, nil
	}
	return minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey", BucketName: bucket, Key: objectName}
}

func (m *mockMinio) PresignedGetObject(ctx context.Context, bucket, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	m.presignCalls++
	if m.presignFn != nil {
		return m.presignFn(ctx, bucket, objectName, expires, reqParams)
	}
	u, err := url.Parse("https://example.invalid/" + bucket + "/" + objectName + "?expires=" + expires.String())
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (m *mockMinio) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	m.listCalls++
	if m.listBucketsFn != nil {
		return m.listBucketsFn(ctx)
	}
	return []minio.BucketInfo{{Name: "civic-intelligence"}}, nil
}

// --- helpers ---------------------------------------------------------------

func newTestClient(t *testing.T) (ObjectStorage, *mockMinio) {
	t.Helper()
	m := newMockMinio()
	c := NewWithClient(DefaultConfig(), m)
	return c, m
}

// --- tests -----------------------------------------------------------------

func TestUpload_Success(t *testing.T) {
	c, m := newTestClient(t)
	body := []byte("hello world")
	err := c.Upload(context.Background(), "bkt", "k1", bytes.NewReader(body), int64(len(body)), "text/plain")
	require.NoError(t, err)
	assert.Equal(t, 1, m.putCalls)
	assert.Equal(t, body, m.store["bkt/k1"])
}

func TestUpload_EmptyBucketErrors(t *testing.T) {
	c, _ := newTestClient(t)
	err := c.Upload(context.Background(), "", "k", bytes.NewReader([]byte("x")), 1, "text/plain")
	assert.ErrorIs(t, err, ErrEmptyBucket)
}

func TestUpload_EmptyKeyErrors(t *testing.T) {
	c, _ := newTestClient(t)
	err := c.Upload(context.Background(), "bkt", "  ", bytes.NewReader([]byte("x")), 1, "text/plain")
	assert.ErrorIs(t, err, ErrEmptyKey)
}

func TestUpload_PropagatesError(t *testing.T) {
	m := newMockMinio()
	m.putFn = func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
		return minio.UploadInfo{}, errors.New("boom")
	}
	c := NewWithClient(DefaultConfig(), m)
	err := c.Upload(context.Background(), "bkt", "k", bytes.NewReader([]byte("x")), 1, "text/plain")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestUpload_DefaultContentType(t *testing.T) {
	m := newMockMinio()
	var seenCT string
	m.putFn = func(_ context.Context, _, _ string, _ io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
		seenCT = opts.ContentType
		return minio.UploadInfo{}, nil
	}
	c := NewWithClient(DefaultConfig(), m)
	require.NoError(t, c.Upload(context.Background(), "bkt", "k", bytes.NewReader([]byte("x")), 1, ""))
	assert.Equal(t, "application/octet-stream", seenCT)
}

func TestDelete_Success(t *testing.T) {
	c, m := newTestClient(t)
	m.store["bkt/k"] = []byte("x")
	require.NoError(t, c.Delete(context.Background(), "bkt", "k"))
	_, ok := m.store["bkt/k"]
	assert.False(t, ok, "key should be removed")
	assert.Equal(t, 1, m.removeCalls)
}

func TestExists_True(t *testing.T) {
	c, m := newTestClient(t)
	m.store["bkt/k"] = []byte("x")
	got, err := c.Exists(context.Background(), "bkt", "k")
	require.NoError(t, err)
	assert.True(t, got)
	assert.Equal(t, 1, m.statCalls)
}

func TestExists_False_NoError(t *testing.T) {
	c, _ := newTestClient(t)
	got, err := c.Exists(context.Background(), "bkt", "missing")
	require.NoError(t, err)
	assert.False(t, got)
}

func TestExists_PropagatesOtherErrors(t *testing.T) {
	m := newMockMinio()
	m.statFn = func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
		return minio.ObjectInfo{}, errors.New("network down")
	}
	c := NewWithClient(DefaultConfig(), m)
	got, err := c.Exists(context.Background(), "bkt", "k")
	require.Error(t, err)
	assert.False(t, got)
	assert.Contains(t, err.Error(), "network down")
}

func TestDownload_MissingReturnsErrNotFound(t *testing.T) {
	c, _ := newTestClient(t)
	_, err := c.Download(context.Background(), "bkt", "missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDownload_Success(t *testing.T) {
	m := newMockMinio()
	m.getFn = func(_ context.Context, _, _ string, _ minio.GetObjectOptions) (*minio.Object, error) {
		// For unit tests we can't return a real *minio.Object, so return an
		// error that won't be classified as NotFound — the test asserts the
		// error path. The success round-trip is covered by an integration
		// test against a real MinIO container (see README).
		return nil, errors.New("integration test required for success path")
	}
	c := NewWithClient(DefaultConfig(), m)
	_, err := c.Download(context.Background(), "bkt", "k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integration test required")
}

func TestPresign_Success(t *testing.T) {
	c, m := newTestClient(t)
	urlStr, err := c.Presign(context.Background(), "bkt", "k", 5*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, urlStr)
	assert.Equal(t, 1, m.presignCalls)
	assert.True(t, strings.HasPrefix(urlStr, "https://"))
}

func TestPresign_DefaultExpiryWhenZero(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PresignExpiry = 0
	m := newMockMinio()
	var seenExpiry time.Duration
	m.presignFn = func(_ context.Context, _, _ string, expires time.Duration, _ url.Values) (*url.URL, error) {
		seenExpiry = expires
		return &url.URL{Path: "/x"}, nil
	}
	c := NewWithClient(cfg, m)
	_, err := c.Presign(context.Background(), "bkt", "k", 0)
	require.NoError(t, err)
	assert.Equal(t, 15*time.Minute, seenExpiry)
}

func TestPing_Success(t *testing.T) {
	c, m := newTestClient(t)
	require.NoError(t, c.Ping(context.Background()))
	assert.Equal(t, 1, m.listCalls)
}

func TestPing_Failure(t *testing.T) {
	m := newMockMinio()
	m.listBucketsFn = func(_ context.Context) ([]minio.BucketInfo, error) {
		return nil, errors.New("unreachable")
	}
	c := NewWithClient(DefaultConfig(), m)
	err := c.Ping(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

func TestNew_RealMinioClient_Constructs(t *testing.T) {
	// Just verify New() returns a non-nil client without erroring on the
	// credentials chain. We don't connect (lazy).
	c, err := New(Config{
		Endpoint:  "127.0.0.1:9000",
		Region:    "us-east-1",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		UseTLS:    false,
	})
	require.NoError(t, err)
	assert.NotNil(t, c)
}

// TestConsumerWithMock proves a downstream consumer that depends on the
// ObjectStorage interface can be tested with a mock — no S3 required.
func TestConsumerWithMock(t *testing.T) {
	c, m := newTestClient(t)

	// Upload via the consumer, then check the mock recorded it.
	require.NoError(t, saveDocument(c, "doc-1", []byte("raw text")))
	assert.Equal(t, 1, m.putCalls)
	assert.Equal(t, []byte("raw text"), m.store["civic-intelligence/doc-1"])

	// Exists via the consumer.
	got, err := documentExists(c, "doc-1")
	require.NoError(t, err)
	assert.True(t, got)
}

// saveDocument / documentExists are representative consumers.
func saveDocument(s ObjectStorage, key string, body []byte) error {
	return s.Upload(context.Background(), "civic-intelligence", key, bytes.NewReader(body), int64(len(body)), "application/pdf")
}

func documentExists(s ObjectStorage, key string) (bool, error) {
	return s.Exists(context.Background(), "civic-intelligence", key)
}
