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

func (t taskFirst) Commit(context.Context) error {
	log.Println("first commit")
	return nil
}

func (t taskFirst) Rollback(context.Context) error {
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

func (t taskSecond) Commit(context.Context) error {
	return errors.New("second commit failed")
}

func (t taskSecond) Rollback(context.Context) error {
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

func (t taskPanic) Commit(context.Context) error {
	panic("3 panic commit")
	return nil
}

func (t taskPanic) Rollback(context.Context) error {
	panic("3 panic rollback")
	return nil
}

func TestSafeTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.NewLogger())
	st := SafeTask(taskPanic{})
	t.Log(NewExecutor(st).Run(ctx))
}

func TestRetryTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.NewLogger())
	t.Log(NewExecutor(RetryTask(taskSecond{}, WithAttempt(3))).Run(ctx))
	t.Log(NewExecutor(RetryTask(taskSecond{}, WithPolicy(PolicyRevert), WithAttempt(3))).Run(ctx))
}

func TestParallelTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.NewLogger())
	st := ParallelTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(NewExecutor(st).Run(ctx))
}

func TestPipelineTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.NewLogger())
	st := PipelineTask(WithTasks(taskFirst{}, taskSecond{}))
	t.Log(NewExecutor(st).Run(ctx))
}

func TestSimpleTask(t *testing.T) {
	f := NewFuncTask(UI)
	t.Log(f.ID(), f.Name())
}

func UI(context.Context) error {
	return nil
}
