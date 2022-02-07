package taskflow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"
)

type TestTaskOne struct {
}

func (t TestTaskOne) ID() string {
	return "1"
}

func (t TestTaskOne) Name() string {
	return "ONE"
}

func (t TestTaskOne) Policy() Policy {
	return PolicyRevert
}

func (t TestTaskOne) Commit(ctx context.Context) error {
	tt := time.NewTimer(20 * time.Second)
	select {
	case <-tt.C:
		fmt.Println("task one Commit")
		tt.Stop()
		return nil
	case <-ctx.Done():
		tt.Stop()
		return ctx.Err()
	}
}

func (t TestTaskOne) Rollback(ctx context.Context) error {
	fmt.Println("task one Rollback")
	return nil
}

type TestTaskTwo struct {
}

func (t TestTaskTwo) ID() string {
	return "2"
}

func (t TestTaskTwo) Name() string {
	return "TWO"
}

func (t TestTaskTwo) Policy() Policy {
	return PolicyRevert
}

func (t TestTaskTwo) Commit(ctx context.Context) error {
	panic("op")
	fmt.Println("task two Commit")
	return errors.New("failed")
}

func (t TestTaskTwo) Rollback(ctx context.Context) error {
	fmt.Println("task two Rollback")
	return errors.New("failed")
}

type TestNotify struct {
}

func (t TestNotify) Event(ctx context.Context, format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (t TestNotify) Notify(ctx context.Context, name string, progress float32) {
	log.Printf("%s %f", name, progress)
}

func TestPipelineFlow(t *testing.T) {
	pf := NewPipelineFlow(WithPolicy(PolicyRevert)).Add(TestTaskOne{}, SafeTask(TestTaskTwo{}))
	t.Log(pf.Run(context.Background()))
}

func TestSpawnFlow(t *testing.T) {
	sf := NewSpawnFlow(WithNotifier(TestNotify{})).Add(TestTaskOne{}, SafeTask(TestTaskTwo{}))
	t.Log(sf.Run(context.Background()))
}
