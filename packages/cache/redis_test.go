package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---------------------------------------------------------------

// newTestClient spins up a miniredis and returns a Client bound to it plus
// a teardown func. miniredis is an in-process Redis server — no Docker, no
// network — so tests are fast and deterministic.
func newTestClient(t *testing.T) (Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := FromClient(cli)
	return c, func() {
		_ = cli.Close()
		mr.Close()
	}
}

// sample is a JSON-friendly struct used by Get/Set tests.
type sample struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
	When  time.Time `json:"when"`
}

// --- tests -----------------------------------------------------------------

func TestSetGet_JSONRoundTrip(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	want := sample{ID: "b-1", Count: 42, When: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)}
	require.NoError(t, c.Set(ctx, "bill:1", want, 0))

	var got sample
	require.NoError(t, c.Get(ctx, "bill:1", &got))
	assert.Equal(t, want, got)
}

func TestGet_MissingKeyReturnsRedisNil(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	var s sample
	err := c.Get(ctx, "does-not-exist", &s)
	assert.True(t, errors.Is(err, redis.Nil), "expected redis.Nil, got %v", err)
}

func TestGet_EmptyKeyErrors(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	var s sample
	err := c.Get(context.Background(), "", &s)
	assert.ErrorIs(t, err, ErrEmptyKey)
}

func TestGet_NilDestErrors(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	// First populate so we reach the unmarshal step.
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", sample{ID: "x"}, 0))
	err := c.Get(ctx, "k", nil)
	assert.ErrorIs(t, err, ErrNilDest)
}

func TestSet_EmptyKeyErrors(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	err := c.Set(context.Background(), "", "v", time.Second)
	assert.ErrorIs(t, err, ErrEmptyKey)
}

func TestSet_DefaultTTL(t *testing.T) {
	// Verify that ttl==0 falls back to DefaultTTL.
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := DefaultConfig()
	cfg.DefaultTTL = 7 * time.Second
	c := &redisClient{
		cfg:    cfg,
		client: cli,
	}

	require.NoError(t, c.Set(context.Background(), "k", "v", 0))

	ttl := mr.TTL("k")
	assert.Greater(t, ttl, time.Duration(0), "key should have a TTL")
	assert.LessOrEqual(t, ttl, 7*time.Second)
}

func TestSet_PerEntryTTLOverridesDefault(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := DefaultConfig()
	cfg.DefaultTTL = 7 * time.Second
	c := &redisClient{cfg: cfg, client: cli}

	require.NoError(t, c.Set(context.Background(), "k", "v", 30*time.Second))
	ttl := mr.TTL("k")
	assert.Greater(t, ttl, 7*time.Second, "per-entry TTL should win")
}

func TestSet_JSONEncodingShape(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", sample{ID: "x", Count: 1}, 0))

	raw, err := c.GetRaw(ctx, "k")
	require.NoError(t, err)

	// The stored value must be valid JSON of the expected shape — proves
	// Set did not store e.g. a Go stringification.
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	assert.Equal(t, "x", m["id"])
}

func TestSetRaw_GetRaw_StringsUnchanged(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	require.NoError(t, c.SetRaw(ctx, "k", "plain-string", time.Minute))
	got, err := c.GetRaw(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "plain-string", got)
}

func TestDelete_RemovesKey(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k", "v", 0))

	n, err := c.Delete(ctx, "k", "missing")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "should report 1 deleted key (the missing one is a no-op)")

	exists, err := c.Exists(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestDelete_EmptyKeyListIsNoop(t *testing.T) {
	c, done := newTestClient(t)
	defer done()
	n, err := c.Delete(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestExists_CountsMatchingKeys(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "a", 1, 0))
	require.NoError(t, c.Set(ctx, "b", 2, 0))

	n, err := c.Exists(ctx, "a", "b", "missing")
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestPing_Healthy(t *testing.T) {
	c, done := newTestClient(t)
	defer done()
	assert.NoError(t, c.Ping(context.Background()))
}

func TestHealth_Healthy(t *testing.T) {
	c, done := newTestClient(t)
	defer done()
	assert.NoError(t, c.Health(context.Background()))
}

func TestHealth_FailsWhenServerDown(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := FromClient(cli)
	mr.Close() // kill the server

	err = c.Health(context.Background())
	assert.Error(t, err, "health should fail when Redis is unreachable")
}

func TestSet_MarshalErrorPropagates(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	// chan cannot be JSON-marshalled → Set must surface the error.
	err := c.Set(context.Background(), "k", make(chan int), 0)
	assert.Error(t, err)
}

func TestGet_UnmarshalErrorPropagates(t *testing.T) {
	c, done := newTestClient(t)
	defer done()

	ctx := context.Background()
	// Put a string that isn't valid JSON for the target type.
	require.NoError(t, c.SetRaw(ctx, "k", `{"id":"x","count":"not-a-number"}`, 0))

	var s sample
	err := c.Get(ctx, "k", &s)
	assert.Error(t, err)
}

// --- mock-based tests (interface only, no miniredis) -----------------------

// mockClient is a hand-rolled Client used to prove the interface is
// mockable without any external mock library. It records calls so tests
// can assert on them.
type mockClient struct {
	getFn      func(ctx context.Context, key string, dest any) error
	setFn      func(ctx context.Context, key string, value any, ttl time.Duration) error
	getRawFn   func(ctx context.Context, key string) (string, error)
	setRawFn   func(ctx context.Context, key, value string, ttl time.Duration) error
	deleteFn   func(ctx context.Context, keys ...string) (int64, error)
	existsFn   func(ctx context.Context, keys ...string) (int64, error)
	pingFn     func(ctx context.Context) error
	healthFn   func(ctx context.Context) error
	closeFn    func() error

	getCalls    int
	setCalls    int
	deleteCalls int
	existsCalls int
	pingCalls   int
	healthCalls int
}

func (m *mockClient) Get(ctx context.Context, key string, dest any) error {
	m.getCalls++
	return m.getFn(ctx, key, dest)
}
func (m *mockClient) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	m.setCalls++
	return m.setFn(ctx, key, value, ttl)
}
func (m *mockClient) GetRaw(ctx context.Context, key string) (string, error) {
	return m.getRawFn(ctx, key)
}
func (m *mockClient) SetRaw(ctx context.Context, key, value string, ttl time.Duration) error {
	return m.setRawFn(ctx, key, value, ttl)
}
func (m *mockClient) Delete(ctx context.Context, keys ...string) (int64, error) {
	m.deleteCalls++
	return m.deleteFn(ctx, keys...)
}
func (m *mockClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	m.existsCalls++
	return m.existsFn(ctx, keys...)
}
func (m *mockClient) Ping(ctx context.Context) error {
	m.pingCalls++
	return m.pingFn(ctx)
}
func (m *mockClient) Health(ctx context.Context) error {
	m.healthCalls++
	return m.healthFn(ctx)
}
func (m *mockClient) Close() error { return m.closeFn() }

// TestConsumerWithMock proves a downstream consumer that depends on the
// cache.Client interface can be tested with a mock — no Redis required.
func TestConsumerWithMock(t *testing.T) {
	var mc mockClient
	mc.getFn = func(_ context.Context, _ string, dest any) error {
		// Pretend the cached value is this struct.
		*(dest.(*sample)) = sample{ID: "mocked", Count: 99}
		return nil
	}
	mc.setFn = func(_ context.Context, _ string, _ any, _ time.Duration) error { return nil }

	got, err := fetchWithCache(context.Background(), &mc, "bill:1")
	require.NoError(t, err)
	assert.Equal(t, "mocked", got.ID)
	assert.Equal(t, 1, mc.getCalls)
	assert.Equal(t, 0, mc.setCalls, "cache hit → Set should NOT be called")
}

// fetchWithCache is a representative consumer: look up a sample by key,
// fall back to a (mocked) DB and write through on miss.
func fetchWithCache(ctx context.Context, c Client, key string) (sample, error) {
	var s sample
	if err := c.Get(ctx, key, &s); err == nil {
		return s, nil
	} else if !errors.Is(err, redis.Nil) {
		return sample{}, err
	}
	// Cache miss → "DB" lookup.
	s = sample{ID: "from-db", Count: 1}
	_ = c.Set(ctx, key, s, 0)
	return s, nil
}
