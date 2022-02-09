package workflow

import (
	"context"

	"go.uber.org/zap"

	"github.com/crochee/workflow/logger"
)

type defaultExecutor struct {
	t Task
}

func NewExecutor(task Task) Executor {
	return &defaultExecutor{t: task}
}

func (d *defaultExecutor) Run(ctx context.Context) error {
	err := d.t.Commit(ctx)
	if err == nil {
		return nil
	}
	logger.From(ctx).Warn("commit failed", zap.Error(err))
	return d.t.Rollback(ctx)
}
