package logger

import (
	"context"
	"log/slog"
	"os"
)

type slogLogger struct {
	logger *slog.Logger
}

func NewSlogLogger(cfg *Config) (Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: mapLevelToSlog(cfg.Level),
	}

	switch cfg.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(output, opts)
	case FormatText, FormatConsole:
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return &slogLogger{
		logger: slog.New(handler),
	}, nil
}

func New() (Logger, error) {
	return NewSlogLogger(DefaultConfig())
}

func NewDevelopment() (Logger, error) {
	return NewSlogLogger(DevelopmentConfig())
}

func (s *slogLogger) Debug(msg string, fields ...Field) {
	s.logger.Debug(msg, convertToSlogAttrs(fields)...)
}

func (s *slogLogger) Info(msg string, fields ...Field) {
	s.logger.Info(msg, convertToSlogAttrs(fields)...)
}

func (s *slogLogger) Warn(msg string, fields ...Field) {
	s.logger.Warn(msg, convertToSlogAttrs(fields)...)
}

func (s *slogLogger) Error(msg string, fields ...Field) {
	s.logger.Error(msg, convertToSlogAttrs(fields)...)
}

func (s *slogLogger) With(fields ...Field) Logger {
	return &slogLogger{
		logger: s.logger.With(convertToSlogAttrs(fields)...),
	}
}

func (s *slogLogger) WithContext(ctx context.Context) Logger {
	return &slogLogger{
		logger: s.logger,
	}
}

func (s *slogLogger) Sync() error {
	return nil
}

func convertToSlogAttrs(fields []Field) []any {
	attrs := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		attrs = append(attrs, f.Key, f.Value)
	}
	return attrs
}

func mapLevelToSlog(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
