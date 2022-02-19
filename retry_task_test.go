package workflow

import (
	"context"
	"testing"

	"github.com/crochee/lirity/logger"
)

func TestRetryTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	t.Log(Execute(ctx, RetryTask(taskSecond{}, WithAttempt(3))))
	t.Log(Execute(ctx, RetryTask(taskSecond{}, WithPolicy(PolicyRevert), WithAttempt(3))))
}
