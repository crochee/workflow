package workflow

import (
	"context"
	"testing"

	"github.com/crochee/lirity/logger"
)

func TestParallelTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	st := ParallelTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(Execute(ctx, st))
}

func TestPipelineTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	st := PipelineTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(Execute(ctx, st))
}
