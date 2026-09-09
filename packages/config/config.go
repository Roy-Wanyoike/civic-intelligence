// Package config provides a minimal environment-variable loader. It is a
// thin shim over struct tags so that each service can define its own config
// struct without depending on a third-party envconfig library.
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Load reads environment variables into the provided struct. Each exported
// field of the struct may carry a tag of the form:
//
//	type Server struct {
//	    Port int    `env:"PORT" default:"8080"`
//	    Host string `env:"HOST" default:"0.0.0.0"`
//	    Timeout time.Duration `env:"TIMEOUT" default:"30s"`
//	}
//
// Load returns an error if a field tagged `env:"X,required"` has no value
// and no default. Only basic kinds (string, int, int64, bool, float64,
// time.Duration) are supported; nested structs are descended into.
func Load(dst any) error {
	return loadInto(reflect.ValueOf(dst), "")
}

func loadInto(v reflect.Value, prefix string) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("config: destination must be non-nil pointer")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config: destination must be a struct, got %s", v.Kind())
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fv := v.Field(i)
		tag := field.Tag.Get("env")
		if tag == "" && fv.Kind() == reflect.Struct {
			// descend into nested struct
			if err := loadInto(fv, prefix); err != nil {
				return err
			}
			continue
		}
		if tag == "" {
			continue
		}
		parts := strings.Split(tag, ",")
		envName := parts[0]
		required := false
		for _, p := range parts[1:] {
			if p == "required" {
				required = true
			}
		}
		defVal := field.Tag.Get("default")

		raw, present := os.LookupEnv(envName)
		if !present {
			if defVal != "" {
				raw = defVal
			} else if required {
				return fmt.Errorf("config: required env var %s not set", envName)
			} else {
				continue
			}
		}
		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("config: env %s: %w", envName, err)
		}
	}
	return nil
}

// setField assigns the raw string to the reflect.Value, parsing it according
// to the field's kind.
func setField(fv reflect.Value, raw string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special case: time.Duration is also an int64 under the hood.
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return err
			}
			fv.SetInt(int64(d))
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element type %s", fv.Type().Elem())
		}
		parts := splitCSV(raw)
		out := reflect.MakeSlice(fv.Type(), len(parts), len(parts))
		for i, p := range parts {
			out.Index(i).SetString(p)
		}
		fv.Set(out)
	default:
		return fmt.Errorf("unsupported field kind %s", fv.Kind())
	}
	return nil
}

// splitCSV splits a comma- or space-separated string, trimming each piece.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, ",", " ")
	parts := strings.Fields(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p)
	}
	return out
}
