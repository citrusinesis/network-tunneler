package logger

import (
	"context"

	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config *Config `optional:"true"`
}

func NewLogger(lc fx.Lifecycle, p Params) (Logger, error) {
	cfg := p.Config
	if cfg == nil {
		cfg = DefaultConfig()
	}

	logger, err := NewSlogLogger(cfg)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return logger.Sync()
		},
	})

	return logger, nil
}

var Module = fx.Provide(NewLogger)
