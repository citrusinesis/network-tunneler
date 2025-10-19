package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSlogLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		logFunc  func(Logger)
		expected bool
	}{
		{
			name:  "debug level logs debug",
			level: LevelDebug,
			logFunc: func(l Logger) {
				l.Debug("debug message")
			},
			expected: true,
		},
		{
			name:  "info level skips debug",
			level: LevelInfo,
			logFunc: func(l Logger) {
				l.Debug("debug message")
			},
			expected: false,
		},
		{
			name:  "info level logs info",
			level: LevelInfo,
			logFunc: func(l Logger) {
				l.Info("info message")
			},
			expected: true,
		},
		{
			name:  "warn level skips info",
			level: LevelWarn,
			logFunc: func(l Logger) {
				l.Info("info message")
			},
			expected: false,
		},
		{
			name:  "error level logs error",
			level: LevelError,
			logFunc: func(l Logger) {
				l.Error("error message")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			cfg := &Config{
				Level:  tt.level,
				Format: FormatJSON,
				Output: buf,
			}

			logger, err := NewSlogLogger(cfg)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}

			tt.logFunc(logger)

			output := buf.String()
			if tt.expected && output == "" {
				t.Errorf("expected log output, got none")
			}
			if !tt.expected && output != "" {
				t.Errorf("expected no log output, got: %s", output)
			}
		})
	}
}

func TestSlogLogger_Fields(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	}

	logger, err := NewSlogLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("test message",
		String("string_field", "value"),
		Int("int_field", 42),
		Bool("bool_field", true),
	)

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("output missing message: %s", output)
	}
	if !strings.Contains(output, "string_field") {
		t.Errorf("output missing string_field: %s", output)
	}
	if !strings.Contains(output, "int_field") {
		t.Errorf("output missing int_field: %s", output)
	}
	if !strings.Contains(output, "bool_field") {
		t.Errorf("output missing bool_field: %s", output)
	}
}

func TestSlogLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	}

	logger, err := NewSlogLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	childLogger := logger.With(
		String("component", "server"),
		String("request_id", "123"),
	)

	childLogger.Info("handling request")

	output := buf.String()
	if !strings.Contains(output, "component") {
		t.Errorf("output missing component field: %s", output)
	}
	if !strings.Contains(output, "server") {
		t.Errorf("output missing component value: %s", output)
	}
	if !strings.Contains(output, "request_id") {
		t.Errorf("output missing request_id field: %s", output)
	}
}

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		key   string
	}{
		{"String", String("key", "value"), "key"},
		{"Int", Int("key", 42), "key"},
		{"Int64", Int64("key", 42), "key"},
		{"Uint", Uint("key", 42), "key"},
		{"Uint64", Uint64("key", 42), "key"},
		{"Float64", Float64("key", 3.14), "key"},
		{"Bool", Bool("key", true), "key"},
		{"Duration", Duration("key", time.Second), "key"},
		{"Time", Time("key", time.Now()), "key"},
		{"Error", Error(errors.New("test")), "error"},
		{"NamedError", NamedError("custom_error", errors.New("test")), "custom_error"},
		{"Any", Any("key", map[string]string{"a": "b"}), "key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.key {
				t.Errorf("expected key %q, got %q", tt.key, tt.field.Key)
			}
			if tt.field.Value == nil {
				t.Errorf("expected non-nil value")
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Level != LevelInfo {
		t.Errorf("expected level Info, got %s", cfg.Level)
	}
	if cfg.Format != FormatJSON {
		t.Errorf("expected format JSON, got %s", cfg.Format)
	}
	if cfg.Development {
		t.Error("expected Development to be false")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	cfg := DevelopmentConfig()
	if cfg.Level != LevelDebug {
		t.Errorf("expected level Debug, got %s", cfg.Level)
	}
	if cfg.Format != FormatConsole {
		t.Errorf("expected format Console, got %s", cfg.Format)
	}
	if !cfg.Development {
		t.Error("expected Development to be true")
	}
}

func TestNew(t *testing.T) {
	logger, err := New()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestNewDevelopment(t *testing.T) {
	logger, err := NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create development logger: %v", err)
	}
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}
