package graceful

import (
	"context"

	"github.com/robfig/cron/v3"
)

type Task interface {
	Spec() string
	Run()
}

func ScheduleRunner(tasks ...Task) Runner {
	cr := cron.New()

	for _, task := range tasks {
		_, _ = cr.AddFunc(task.Spec(), task.Run)
	}

	return func(ctx context.Context) error {
		cr.Start()
		defer cr.Stop()

		<-ctx.Done()

		return ctx.Err()
	}
}
