package graceful

import (
	"context"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

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
