package workflow

import (
	"context"

	"github.com/crochee/lirity/logger"
	"go.uber.org/zap"
)

type FuncExecutor func(ctx context.Context, task Task) error

func Executor(ctx context.Context, task Task) error {
	err := task.Commit(ctx)
	if err == nil {
		return nil
	}
	logger.From(ctx).Warn("commit failed", zap.Error(err))
	return task.Rollback(ctx)
}
