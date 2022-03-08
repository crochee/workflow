package workflow

import (
	"context"
	"log"
	"testing"

	"github.com/crochee/lirity/logger"
)

type callback struct {
}

func (c callback) Trigger(ctx context.Context, task Task) {
	log.Println(task.ID(), task.Name(), task.Description())
	log.Println(task.State())
}

func TestParallelTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	st := ParallelTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(Execute(ctx, st))
}

func TestPipelineTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	st := PipelineTask(WithTasks(taskFirst{}, taskSecond{}), WithCallbacks(callback{}))
	t.Log(Execute(ctx, st))
}
