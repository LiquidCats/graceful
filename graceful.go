package graceful

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var ErrShutdownBySignal = errors.New("shutdown by signal")

type HttpConfig struct {
	Port         string        `envconfig:"PORT" json:"port" yaml:"port" default:"8080"`
	ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" json:"read_timeout" yaml:"read_timeout" default:"10s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" json:"write_timeout" yaml:"write_timeout" default:"10s"`
}

type Runner func(ctx context.Context) error

func Signals(ctx context.Context) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer signal.Stop(sigs)

	for {
		select {
		case <-sigs:
			return ErrShutdownBySignal
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func ServerRunner(router http.Handler, cfg HttpConfig) Runner {
	return func(ctx context.Context) error {
		group, groupCtx := errgroup.WithContext(ctx)

		server := &http.Server{
			Addr:           net.JoinHostPort("0.0.0.0", cfg.Port),
			Handler:        router,
			ReadTimeout:    cfg.ReadTimeout,
			WriteTimeout:   cfg.WriteTimeout,
			MaxHeaderBytes: http.DefaultMaxHeaderBytes,
		}

		group.Go(func() error {
			return server.ListenAndServe()
		})

		group.Go(func() error {
			<-groupCtx.Done()

			srvCtx, cancel := context.WithTimeout(context.WithoutCancel(groupCtx), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(srvCtx); err != nil {
				return err
			}

			return nil
		})

		return group.Wait()
	}
}

func WaitContext(ctx context.Context, runners ...Runner) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)

	// Start a goroutine for each runner
	for _, r := range runners {
		runner := r
		group.Go(func() error {
			return runner(ctx)
		})
	}

	if err := group.Wait(); errors.Is(err, ErrShutdownBySignal) {
		return nil
	} else {
		return err
	}
}
