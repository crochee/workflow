package workflow

import (
	"context"

	"github.com/crochee/lirity/logger"
	"go.uber.org/zap"
)

type Executor interface {
	Execute(ctx context.Context, task Task) error
}

type FuncExecutor func(ctx context.Context, task Task) error

func (f FuncExecutor) Execute(ctx context.Context, task Task) error {
	return f(ctx, task)
}

func Execute(ctx context.Context, task Task) error {
	err := task.Commit(ctx)
	if err == nil {
		return nil
	}
	logger.From(ctx).Warn("commit failed", zap.Error(err))
	return task.Rollback(ctx)
}
