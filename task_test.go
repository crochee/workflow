package workflow

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/crochee/lirity/logger"
)

type taskFirst struct {
}

func (t taskFirst) ID() string {
	return "1"
}

func (t taskFirst) Name() string {
	return "first"
}

func (t taskFirst) State() (State, error) {
	return Success, nil
}

func (t taskFirst) Description() string {
	return ""
}

func (t taskFirst) CreateTime() time.Time {
	return time.Now()
}

func (t taskFirst) UpdateTime() time.Time {
	return time.Now()
}

func (t taskFirst) Meta() map[string]interface{} {
	return map[string]interface{}{}
}

func (t taskFirst) Commit(ctx context.Context, opts ...TaskOption) error {
	log.Println("first commit")
	return nil
}

func (t taskFirst) Rollback(ctx context.Context, opts ...TaskOption) error {
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

func (t taskSecond) State() (State, error) {
	return Success, nil
}

func (t taskSecond) Description() string {
	return ""
}

func (t taskSecond) CreateTime() time.Time {
	return time.Now()
}

func (t taskSecond) UpdateTime() time.Time {
	return time.Now()
}

func (t taskSecond) Meta() map[string]interface{} {
	return map[string]interface{}{}
}

func (t taskSecond) Commit(ctx context.Context, opts ...TaskOption) error {
	return errors.New("second commit failed")
}

func (t taskSecond) Rollback(ctx context.Context, opts ...TaskOption) error {
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

func (t taskPanic) State() (State, error) {
	return Success, nil
}

func (t taskPanic) Description() string {
	return ""
}

func (t taskPanic) CreateTime() time.Time {
	return time.Now()
}

func (t taskPanic) UpdateTime() time.Time {
	return time.Now()
}

func (t taskPanic) Meta() map[string]interface{} {
	return map[string]interface{}{}
}

func (t taskPanic) Commit(ctx context.Context, opts ...TaskOption) error {
	panic("3 panic commit")
	return nil
}

func (t taskPanic) Rollback(ctx context.Context, opts ...TaskOption) error {
	panic("3 panic rollback")
	return nil
}

func TestSafeTask(t *testing.T) {
	ctx := logger.With(context.Background(), logger.New())
	st := SafeTask(taskPanic{})
	t.Log(Execute(ctx, st))
}

func TestSimpleTask(t *testing.T) {
	f := NewFuncTask(UI)
	t.Log(f.ID(), f.Name())
}

func UI(context.Context) error {
	return nil
}
