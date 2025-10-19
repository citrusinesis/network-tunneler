package logger

import (
	"context"
	"io"
	"time"
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger

	Sync() error
}

type Field struct {
	Key   string
	Value any
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Uint(key string, value uint) Field {
	return Field{Key: key, Value: value}
}

func Uint64(key string, value uint64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

func NamedError(key string, err error) Field {
	return Field{Key: key, Value: err}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Format string

const (
	FormatJSON    Format = "json"
	FormatConsole Format = "console"
	FormatText    Format = "text"
)

type Config struct {
	Level       Level
	Format      Format
	Output      io.Writer
	Development bool
}

func DefaultConfig() *Config {
	return &Config{
		Level:       LevelInfo,
		Format:      FormatJSON,
		Output:      nil,
		Development: false,
	}
}

func DevelopmentConfig() *Config {
	return &Config{
		Level:       LevelDebug,
		Format:      FormatConsole,
		Output:      nil,
		Development: true,
	}
}
