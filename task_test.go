package workflow

import (
	"context"
	"errors"
	"log"
	"testing"

	"github.com/crochee/workflow/logger"
)

type taskFirst struct {
}

func (t taskFirst) ID() string {
	return "1"
}

func (t taskFirst) Name() string {
	return "first"
}

func (t taskFirst) Commit(ctx context.Context) error {
	log.Println("first commit")
	return nil
}

func (t taskFirst) Rollback(ctx context.Context) error {
	log.Println("first rollback")
	return nil
}

type taskSecond struct {
}

func (t taskSecond) ID() string {
	return "2"
}

func (t taskSecond) Name() string {
	return "second"
}

func (t taskSecond) Commit(ctx context.Context) error {
	return errors.New("second commit failed")
}

func (t taskSecond) Rollback(ctx context.Context) error {
	log.Println("second rollback")
	return nil
}

type taskPanic struct {
}

func (t taskPanic) ID() string {
	return "3"
}

func (t taskPanic) Name() string {
	return "panic"
}

func (t taskPanic) Commit(ctx context.Context) error {
	panic("3 panic commit")
	return nil
}

func (t taskPanic) Rollback(ctx context.Context) error {
	panic("3 panic rollback")
	return nil
}

func TestSafeTask(t *testing.T) {
	st := SafeTask(taskPanic{})
	t.Log(DefaultExecutor(st).Run(context.Background()))
}

func TestRetryTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.NewLogger())
	t.Log(DefaultExecutor(RetryTask(taskSecond{}, WithAttempt(3))).Run(ctx))
	t.Log(DefaultExecutor(RetryTask(taskSecond{}, WithPolicy(PolicyRevert), WithAttempt(3))).Run(ctx))
}

func TestParallelTask(t *testing.T) {
	st := ParallelTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(DefaultExecutor(st).Run(context.Background()))
}

func TestPipelineTask(t *testing.T) {
	st := PipelineTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(DefaultExecutor(st).Run(context.Background()))
}
