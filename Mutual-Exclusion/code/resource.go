package mutual

import (
	"fmt"
	"sync"
	"time"
)

var (
	// NOBODY 表示没有赋予任何人
	NOBODY = -1 // TODO: 删除此处内容
	// NOBODY2 表示没有赋予任何人
	NOBODY2 = timestamp{time: -1, process: others}
)

type resource struct {
	occupiedBy2 timestamp      // TODO: 重命名此处属性
	occupiedBy  int            // TODO: 删除此处内容
	procOrder   []int          // TODO: 删除此处内容
	timeOrder   []int          // TODO: 删除此处内容
	occupieds   sync.WaitGroup // TODO: 删除此处内容

	timestamps []timestamp
	times      []time.Time
}

func newResource() *resource {
	return &resource{
		occupiedBy2: NOBODY2,
		occupiedBy:  NOBODY,
	}
}

func (r *resource) occupy2(ts timestamp) { // TODO: 修改方法名称
	if r.occupiedBy2 != NOBODY2 {
		msg := fmt.Sprintf("资源正在被 %s 占据，%s 却想获取资源。", r.occupiedBy2, ts)
		panic(msg)
	}
	r.occupiedBy2 = ts
	r.timestamps = append(r.timestamps, ts)
	r.times = append(r.times, time.Now())
	debugPrintf("~~~ @resource: %s occupied ~~~ ", ts)
}

func (r *resource) release2(ts timestamp) { // TODO: 修改方法名称
	if r.occupiedBy2 != ts {
		msg := fmt.Sprintf("%s 想要释放正在被 P%s 占据的资源。", ts, r.occupiedBy2)
		panic(msg)
	}
	r.occupiedBy2 = NOBODY2
	r.times = append(r.times, time.Now())
	debugPrintf("~~~ @resource: %s released ~~~ ", ts)
}

func (r *resource) report() string {
	occupiedTime := time.Duration(0)
	size := len(r.times)
	for i := 0; i+1 < size; i += 2 {
		occupiedTime += r.times[i+1].Sub(r.times[i])
	}
	totalTime := r.times[size-1].Sub(r.times[0])
	rate := occupiedTime.Nanoseconds() * 10000 / totalTime.Nanoseconds()
	format := "resource 的占用比率为 %02d.%02d%%"
	return fmt.Sprintf(format, rate/100, rate%100)
}

func (r *resource) isSortedOccupied() bool {
	size := len(r.timestamps)
	for i := 1; i < size; i++ {
		if !less(r.timestamps[i-1], r.timestamps[i]) {
			return false
		}
	}
	return true
}

// TODO: 删除以下内容

func (r *resource) occupy(req *request) {
	if r.occupiedBy != NOBODY {
		msg := fmt.Sprintf("资源正在被 P%d 占据，P%d 却想获取资源。", r.occupiedBy, req.process)
		panic(msg)
	}
	r.occupiedBy = req.process
	r.procOrder = append(r.procOrder, req.process)
	r.timeOrder = append(r.timeOrder, req.timestamp)
	debugPrintf("~~~ @resource: %s occupy ~~~ %v", req, r.procOrder[max(0, len(r.procOrder)-6):])
}

func (r *resource) release(req *request) {
	if r.occupiedBy != req.process {
		msg := fmt.Sprintf("P%d 想要释放正在被 P%d 占据的资源。", req.process, r.occupiedBy)
		panic(msg)
	}
	r.occupiedBy = NOBODY

	debugPrintf("~~~ @resource: %s release ~~~ %v", req, r.procOrder[max(0, len(r.procOrder)-6):])

	r.occupieds.Done()
}
