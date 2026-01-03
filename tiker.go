package graceful

import (
	"context"
	"time"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

var ErrTickerFailure = eris.New("ticker failure")

type ticker struct {
	logger zerolog.Logger
}

type TickerOpt func(*ticker)

func WithTickerLogger(logger zerolog.Logger) TickerOpt {
	return func(t *ticker) {
		t.logger = logger
	}
}

func NewTicker(interval time.Duration, runner Runner, opts ...TickerOpt) Runner {
	cfg := &ticker{
		logger: zerolog.Nop(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx context.Context) error {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		cfg.logger.Info().Msg("starting ticker")
		defer cfg.logger.Info().Msg("stopped ticker")

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if err := runner(ctx); err != nil {
					if eris.Is(err, ErrTickerFailure) {
						return err
					}
					cfg.logger.
						Error().
						Any("error", eris.ToJSON(err, true)).
						Msg("runner failed")
				}
			}
		}
	}
}
