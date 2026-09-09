// Package config provides shared configuration loading for all Go services.
// Each service defines its own config struct with `env:"NAME" default:"value"`
// tags; this package loads from environment variables.
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Load populates a config struct from environment variables using `env:"NAME"`
// and `default:"value"` struct tags. Supported field types: string, int,
// int64, bool, time.Duration.
//
// Example struct:
//
//	type Config struct {
//	    HTTPAddr        string        `env:"HTTP_ADDR" default:":8084"`
//	    ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`
//	}
func Load(cfg interface{}) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("config: Load requires a non-nil pointer")
	}
	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		envName := field.Tag.Get("env")
		defVal := field.Tag.Get("default")
		var raw string
		if envName != "" {
			raw = os.Getenv(envName)
		}
		if raw == "" {
			raw = defVal
		}
		if raw == "" {
			continue
		}
		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("config: field %s: %w", field.Name, err)
		}
	}
	return nil
}

func setField(fv reflect.Value, raw string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Int, reflect.Int64:
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("parse duration %q: %w", raw, err)
			}
			fv.SetInt(int64(d))
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("parse int %q: %w", raw, err)
		}
		fv.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse bool %q: %w", raw, err)
		}
		fv.SetBool(b)
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(raw, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			fv.Set(reflect.ValueOf(parts))
		}
	}
	return nil
}

// GetString reads a string env var with a fallback default.
func GetString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetInt reads an int env var with a fallback default.
func GetInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// GetDuration reads a duration env var with a fallback default.
func GetDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

// GetBool reads a boolean env var with a fallback default.
func GetBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// MustGetString reads a string env var or panics if missing.
func MustGetString(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}
