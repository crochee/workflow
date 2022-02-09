package workflow

import (
	"context"
	"go.uber.org/zap"
	"time"

	"github.com/crochee/workflow/logger"
)

type defaultExecutor struct {
	t Task
}

func DefaultExecutor(task Task) Executor {
	return &defaultExecutor{t: task}
}

func (d *defaultExecutor) Run(ctx context.Context) error {
	err := d.t.Commit(ctx)
	if err == nil {
		return nil
	}
	logger.From(ctx).Warn("", zap.Duration("time", time.Minute), zap.String("service", "90"), zap.Bools("bools", []bool{true, false}), zap.Error(err))
	return d.t.Rollback(ctx)
}
