// FIXME: verify with go build when Go available
//
// Package cache is a thin, opinionated wrapper around github.com/redis/go-redis/v9.
//
// It provides:
//   - JSON-aware Get/Set (any value is JSON-encoded; Get takes a pointer)
//   - Raw Get/SetRaw for callers that already have bytes
//   - Per-entry TTL (overrides the default if non-zero)
//   - Connection pooling (go-redis's internal pool; configurable)
//   - Health check (Ping + optional INFO)
//   - Exists / Delete
//
// The wrapper deliberately exposes a small surface (a Client struct + an
// interface). All consumers depend on the interface so unit tests can swap
// in miniredis or a mock without touching the production code path.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultTTL is used when a Set call passes ttl == 0.
const DefaultTTL = 5 * time.Minute

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Config holds the connection and pool settings for the Redis client.
type Config struct {
	// Address is the host:port of a single Redis instance. For clusters use
	// NewClusterClient instead (not provided here — add when needed).
	Address string

	// Username / Password (Redis 6+ ACL).
	Username string
	Password string

	// DB is the logical database index (0–15).
	DB int

	// PoolSize is the max number of socket connections. go-redis default is
	// 10 × runtime.NumCPU(); for a sidecar with many goroutines we want this
	// explicit.
	PoolSize int

	// MinIdleConns keeps this many idle connections warm to avoid latency
	// spikes on cold reads.
	MinIdleConns int

	// DialTimeout / ReadTimeout / WriteTimeout bound the network operations.
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// DefaultTTL is the TTL applied to Set calls that pass ttl == 0.
	DefaultTTL time.Duration
}

// DefaultConfig returns a Config that works against a local Redis with
// sensible production-ish defaults. Override individual fields as needed.
func DefaultConfig() Config {
	return Config{
		Address:      "127.0.0.1:6379",
		DB:           0,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		DefaultTTL:   DefaultTTL,
	}
}

// ---------------------------------------------------------------------------
// Client interface + concrete implementation
// ---------------------------------------------------------------------------

// Client is the interface every consumer depends on. It is small enough to
// mock with a hand-rolled struct (see redis_test.go) and matches the
// underlying go-redis UniversalCommands subset we actually use.
type Client interface {
	// Get retrieves a JSON-encoded value and unmarshals it into dest. If the
	// key does not exist the error is redis.Nil (use errors.Is to detect).
	Get(ctx context.Context, key string, dest any) error

	// Set JSON-encodes value and stores it with the given TTL. If ttl == 0
	// the client's DefaultTTL is used.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// GetRaw returns the raw string value. redis.Nil if missing.
	GetRaw(ctx context.Context, key string) (string, error)

	// SetRaw stores a pre-encoded string value with the given TTL.
	SetRaw(ctx context.Context, key, value string, ttl time.Duration) error

	// Delete removes one or more keys. Returns the count of keys deleted.
	Delete(ctx context.Context, keys ...string) (int64, error)

	// Exists returns the count of keys that exist (out of the given set).
	Exists(ctx context.Context, keys ...string) (int64, error)

	// Ping is a lightweight liveness check (returns PONG).
	Ping(ctx context.Context) error

	// Health checks liveness + connectivity beyond PING (issues a dummy
	// GET on a sentinel key). Used by /healthz endpoints.
	Health(ctx context.Context) error

	// Close releases the connection pool.
	Close() error
}

// Compile-time assertion that *redisClient implements Client.
var _ Client = (*redisClient)(nil)

// redisClient wraps *redis.Client.
type redisClient struct {
	cfg    Config
	client *redis.Client
}

// New constructs a Client from Config. The connection is lazy — Ping on
// first use. Call Ping(ctx) explicitly at startup if you want fail-fast.
func New(cfg Config) Client {
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:6379"
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 20
	}
	if cfg.MinIdleConns < 0 {
		cfg.MinIdleConns = 5
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 3 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 2 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 2 * time.Second
	}
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = DefaultTTL
	}

	cli := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	return &redisClient{cfg: cfg, client: cli}
}

// FromClient lets tests construct a Client wrapping a pre-built *redis.Client
// (e.g. a miniredis-backed client). Production code should use New().
func FromClient(cli *redis.Client) Client {
	return &redisClient{cfg: DefaultConfig(), client: cli}
}

// ---------------------------------------------------------------------------
// Get / Set (JSON)
// ---------------------------------------------------------------------------

// Get implements Client.
func (c *redisClient) Get(ctx context.Context, key string, dest any) error {
	if key == "" {
		return ErrEmptyKey
	}
	if dest == nil {
		return ErrNilDest
	}
	raw, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return wrapRedisErr(err)
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("cache: unmarshal %q: %w", key, err)
	}
	return nil
}

// Set implements Client.
func (c *redisClient) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if key == "" {
		return ErrEmptyKey
	}
	body, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal %q: %w", key, err)
	}
	if ttl <= 0 {
		ttl = c.cfg.DefaultTTL
	}
	if err := c.client.Set(ctx, key, body, ttl).Err(); err != nil {
		return wrapRedisErr(err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Get / Set raw
// ---------------------------------------------------------------------------

// GetRaw implements Client.
func (c *redisClient) GetRaw(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", wrapRedisErr(err)
	}
	return val, nil
}

// SetRaw implements Client.
func (c *redisClient) SetRaw(ctx context.Context, key, value string, ttl time.Duration) error {
	if key == "" {
		return ErrEmptyKey
	}
	if ttl <= 0 {
		ttl = c.cfg.DefaultTTL
	}
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return wrapRedisErr(err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Delete / Exists / Ping / Health / Close
// ---------------------------------------------------------------------------

// Delete implements Client.
func (c *redisClient) Delete(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	n, err := c.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, wrapRedisErr(err)
	}
	return n, nil
}

// Exists implements Client.
func (c *redisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	n, err := c.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, wrapRedisErr(err)
	}
	return n, nil
}

// Ping implements Client.
func (c *redisClient) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: ping: %w", err)
	}
	return nil
}

// Health implements Client. Beyond PING, it issues a GET on a sentinel key
// that is guaranteed to return redis.Nil (key never set). This exercises the
// full read path so a half-broken connection (e.g. AUTH failed mid-session)
// is caught here rather than at the first real Get.
func (c *redisClient) Health(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: health: ping: %w", err)
	}
	// Sentinel key — never written. The expected result is redis.Nil, which
	// means the read path works. Any other error means the connection is
	// unhealthy.
	sentinel := "__civic_health_probe__"
	if err := c.client.Get(ctx, sentinel).Err(); !errors.Is(err, redis.Nil) {
		return fmt.Errorf("cache: health: get: %w", err)
	}
	return nil
}

// Close implements Client.
func (c *redisClient) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("cache: close: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// Sentinel errors. Use errors.Is to detect them.
var (
	ErrEmptyKey = errors.New("cache: empty key")
	ErrNilDest  = errors.New("cache: nil destination")
)

// wrapRedisErr translates go-redis errors into sentinel forms. redis.Nil is
// passed through unchanged so callers can detect "not found".
func wrapRedisErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, redis.Nil) {
		return err // pass through — callers check this
	}
	return fmt.Errorf("cache: redis: %w", err)
}
