package workflow

import (
	"context"
	"testing"
	"time"
)

func TestRetryTask(t *testing.T) {
	f := NewFunc(UI)

	t.Log(RetryTask(f, WithAttempt(3)).Execute(context.Background(), nil))
	t.Log(RetryTask(f, WithAttempt(3), WithInterval(time.Second)).Execute(context.Background(), nil))
}
