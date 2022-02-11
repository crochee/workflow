package workflow

import (
	"context"
	"testing"

	"github.com/crochee/lirity/logger"
)

func TestRetryTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	t.Log(Executor(ctx, RetryTask(taskSecond{}, WithAttempt(3))))
	t.Log(Executor(ctx, RetryTask(taskSecond{}, WithPolicy(PolicyRevert), WithAttempt(3))))
}
