package taskflow

import (
	"context"
)

type serialEngine struct {
}

func (s serialEngine) Notify(ctx context.Context, progress float32) error {
	panic("implement me")
}

func (s serialEngine) Statistics() map[string]interface{} {
	panic("implement me")
}

func (s serialEngine) Reset() {
	panic("implement me")
}

func (s serialEngine) Prepare() {
	panic("implement me")
}

func (s serialEngine) Run(ctx context.Context) error {
	panic("implement me")
}

func (s serialEngine) Suspend() {
	panic("implement me")
}

//
//type defaultExecutor struct {
//	id         uint64
//	schedule   Scheduler
//	wrappedJob Job
//	job        Job
//	interval   time.Duration
//	attempts   int
//	cancel     context.CancelFunc
//}
//
//func (d *defaultExecutor) Next(t time.Time) time.Time {
//	return d.schedule.Next(t)
//}
//
//func (d *defaultExecutor) Run(ctx context.Context) error {
//	ctx, d.cancel = context.WithCancel(ctx)
//	err := d.job.Do(ctx)
//	if err != nil { // 如果任务失败的话，重试
//		var (
//			attempts  int
//			breakFlag bool
//		)
//		backOff := d.newBackOff() // 退避算法 保证时间间隔为指数级增长
//		currentInterval := 0 * time.Millisecond
//		t := time.NewTimer(currentInterval)
//		for {
//			select {
//			case <-t.C:
//				shouldRetry := attempts < d.attempts
//				if !shouldRetry {
//					t.Stop()
//					breakFlag = true
//					break
//				}
//				if err = d.job.Redo(ctx); err == nil {
//					t.Stop()
//					return nil
//				}
//				// 计算下一次
//				currentInterval = backOff.NextBackOff()
//				attempts++
//				// 定时器重置
//				t.Reset(currentInterval)
//			case <-ctx.Done():
//				t.Stop()
//				breakFlag = true
//			}
//			if breakFlag {
//				break
//			}
//		}
//	}
//	if err == nil {
//		return nil
//	}
//	// 任务执行不成功, 回调回滚方法
//	return d.job.Revert(ctx)
//}
//
//func (d *defaultExecutor) Stop() error {
//	if d.cancel == nil {
//		return nil
//	}
//	d.cancel()
//	return nil
//}
//
//func (d *defaultExecutor) newBackOff() Next {
//	if d.attempts < 2 || d.interval <= 0 {
//		return &backoff.ZeroBackOff{}
//	}
//
//	b := backoff.NewExponentialBackOff()
//	b.InitialInterval = d.interval
//
//	// calculate the multiplier for the given number of attempts
//	// so that applying the multiplier for the given number of attempts will not exceed 2 times the initial interval
//	// it allows to control the progression along the attempts
//	b.Multiplier = math.Pow(2, 1/float64(d.attempts-1))
//
//	// according to docs, b.Reset() must be called before using
//	b.Reset()
//	return b
//}
