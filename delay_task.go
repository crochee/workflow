package workflow

import (
	"container/list"
	"context"
	"time"
)

type DelayTask struct {
	Delay time.Duration
	Task
}

// timeWheel 时间轮
type timeWheel struct {
	interval time.Duration // 指针每隔多久往前移动一格
	ticker   *time.Ticker
	slots    []*list.List // 时间轮槽
	// key: 定时器唯一标识 value: 定时器所在的槽, 主要用于删除定时器, 不会出现并发读写，不加锁直接访问
	timer             map[string]int
	cur               int             // 当前指针指向哪一个槽
	slotSum           int             // 槽数量
	callback          []Callback      // 定时器回调函数
	addTaskChannel    chan *taskEntry // 新增任务channel
	removeTaskChannel chan string     // 删除任务channel
	stopChannel       chan struct{}   // 停止定时器channel
}

type taskEntry struct {
	DelayTask
	circle  int
	removed bool
}

func NewTimeWheel(interval time.Duration, slotNum int) *timeWheel {
	tw := &timeWheel{
		interval:          interval,
		slots:             make([]*list.List, slotNum),
		timer:             make(map[string]int),
		cur:               -1,
		slotSum:           slotNum,
		callback:          nil,
		addTaskChannel:    make(chan *taskEntry),
		removeTaskChannel: make(chan string),
		stopChannel:       make(chan struct{}),
	}
	for i := 0; i < tw.slotSum; i++ {
		tw.slots[i] = list.New()
	}
	return tw
}

func (tw *timeWheel) Start(ctx context.Context) {
	tw.ticker = time.NewTicker(tw.interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				tw.ticker.Stop()
				return
			case <-tw.ticker.C:
				tw.handler(ctx)
			case task := <-tw.addTaskChannel:
				tw.addTask(task)
			case id := <-tw.removeTaskChannel:
				tw.removeTask(id)
			case <-tw.stopChannel:
				tw.ticker.Stop()
				return
			}
		}
	}()
}

// Stop 停止时间轮
func (tw *timeWheel) Stop() {
	close(tw.stopChannel)
}

// AddTimer 添加定时器 key为定时器唯一标识
func (tw *timeWheel) AddTimer(task DelayTask) {
	if task.Delay < 0 {
		return
	}
	tw.addTaskChannel <- &taskEntry{
		DelayTask: task,
	}
}

// RemoveTimer 删除定时器 id为添加定时器时传递的定时器唯一标识
func (tw *timeWheel) RemoveTimer(id string) {
	if id == "" {
		return
	}
	tw.removeTaskChannel <- id
}

func (tw *timeWheel) handler(ctx context.Context) {
	tw.cur = (tw.cur + 1) % tw.slotSum
	l := tw.slots[tw.cur]
	tw.scanAndRunTask(ctx, l)
}

// 新增任务到链表中
func (tw *timeWheel) addTask(task *taskEntry) {
	pos, circle := tw.getPositionAndCircle(task.Delay)
	task.circle = circle
	tw.slots[pos].PushBack(task)
	tw.timer[task.ID()] = pos
}

// // 从链表中删除任务
func (tw *timeWheel) removeTask(id string) {
	// 获取定时器所在的槽
	position, found := tw.timer[id]
	if !found {
		return
	}
	// 获取槽指向的链表
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task, ok := e.Value.(*taskEntry)
		if !ok {
			next := e.Next()
			l.Remove(e)
			e = next
			continue
		}
		if task.ID() == id {
			delete(tw.timer, id)
			l.Remove(e)
		}
		e = e.Next()
	}
}

// 扫描链表中过期定时器, 并执行回调函数
func (tw *timeWheel) scanAndRunTask(ctx context.Context, l *list.List) {
	for e := l.Front(); e != nil; {
		task, ok := e.Value.(*taskEntry)
		if !ok {
			next := e.Next()
			l.Remove(e)
			e = next
			continue
		}
		if task.removed {
			next := e.Next()
			l.Remove(e)
			e = next
			continue
		}
		if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		}
		go tw.callbacks(ctx, task.Task)
		next := e.Next()
		l.Remove(e)
		delete(tw.timer, task.ID())
		e = next
	}
}

func (tw *timeWheel) callbacks(ctx context.Context, task Task) {
	for _, callback := range tw.callback {
		callback.Trigger(ctx, task, nil, nil)
	}
}

// 获取定时器在槽中的位置, 时间轮需要转动的圈数
func (tw *timeWheel) getPositionAndCircle(d time.Duration) (pos int, circle int) {
	steps := int(d / tw.interval)
	pos = (tw.cur + steps) % tw.slotSum
	circle = (steps - 1) / tw.slotSum
	return
}
