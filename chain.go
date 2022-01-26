package taskflow

// JobWrapper decorates the given Job with some behavior.
type JobWrapper func(Job) Job

// Chain is a sequence of JobWrappers that decorates submitted jobs with
// cross-cutting behaviors like logging or synchronization.
type Chain struct {
	wrappers []JobWrapper
}

// NewChain returns a Chain consisting of the given JobWrappers.
func NewChain(c ...JobWrapper) Chain {
	return Chain{c}
}

// Then decorates the given job with all JobWrappers in the chain.
//
// This:
//     NewChain(m1, m2, m3).Then(job)
// is equivalent to:
//     m1(m2(m3(job)))
func (c Chain) Then(j Job) Job {
	for i := range c.wrappers {
		j = c.wrappers[len(c.wrappers)-i-1](j)
	}
	return j
}

//// Recover panics in wrapped jobs and log them with the provided logger.
//func Recover(logger log.Interface) JobWrapper {
//	return func(j Job) Job {
//		return FuncJob(func() {
//			defer func() {
//				if r := recover(); r != nil {
//					const size = 64 << 10
//					buf := make([]byte, size)
//					buf = buf[:runtime.Stack(buf, false)]
//					err, ok := r.(error)
//					if !ok {
//						err = fmt.Errorf("%v", r)
//					}
//					logger.Error(err, "panic", "stack", "...\n"+string(buf))
//				}
//			}()
//			j.Run()
//		})
//	}
//}
