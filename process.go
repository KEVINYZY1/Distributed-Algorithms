package mutual

import (
	"math/rand"
	"sync"
	"time"
)

type process struct {
	rwmu             *sync.RWMutex
	me               int
	clock            *clock
	chans            []chan message
	requestQueue     []*request
	sentTime         []int         // 最近一次给别的 process 发送的消息，所携带的最后时间
	receiveTime      []int         // 最近一次从别的 process 收到的消息，所携带的最后时间
	minReceiveTime   int           // lastReceiveTime 中的最小值
	toCheckRule5Chan chan struct{} // 每次收到 message 后，都靠这个 chan 来通知检查此 process 是否已经满足 rule 5，以便决定是否占有 resource

	occupying *sync.Mutex
}

func newProcess(me int, chans []chan message) *process {
	p := &process{
		me:               me,
		clock:            newClock(),
		chans:            chans,
		requestQueue:     make([]*request, 0, 1024),
		sentTime:         make([]int, len(chans)),
		receiveTime:      make([]int, len(chans)),
		minReceiveTime:   0,
		toCheckRule5Chan: make(chan struct{}, 1),
	}

	return p
}

func (p *process) occupy() {
	// 可以连续占用资源，但是不能重复占用资源
	p.occupying.Lock()

	rsc.occupy(p.me)

	// 经过一段时间，就释放资源
	go func(p *process) {
		timeout := time.Duration(100+rand.Intn(900)) * time.Millisecond
		time.Sleep(timeout)
		p.release()
		p.occupying.Unlock()
	}(p)
}

func (p *process) release() {
	r := p.requestQueue[0]

	rsc.release(p.me)

	p.delete()

	// TODO: 这算不算一个 event 呢
	p.clock.tick()

	p.messaging(releaseResource, r)
}

func (p *process) request() {
	r := &request{
		time:    p.clock.getTime(),
		process: p.me,
	}

	p.append(r)

	// TODO: 这算不算一个 event 呢
	p.clock.tick()

	p.messaging(requestResource, r)
}

func (p *process) messaging(mt msgType, r *request) {
	for i, ch := range p.chans {
		if i == p.me {
			continue
		}
		ch <- message{
			msgType:  mt,
			time:     p.clock.getTime(),
			senderID: p.me,
			request:  r,
		}
		// sending 是一个 event
		// 所以，发送完成后，需要 clock.tick()
		p.clock.tick()
	}
}

func (p *process) receiveLoop() {
	msgChan := p.chans[p.me]
	for {
		msg := <-msgChan

		p.rwmu.Lock()

		// 接收到了一个新的消息
		// 根据 IR2
		// process 的 clock 需要根据 msg.time 进行更新
		// 无论 msg 是什么类型的消息
		p.clock.update(msg.time)
		p.receiveTime[msg.senderID] = msg.time
		p.updateMinReceiveTime()

		switch msg.msgType {
		case requestResource:
			p.append(msg.request)
		case releaseResource:
			p.delete()
		}

		p.toCheckRule5Chan <- struct{}{}

		p.rwmu.Unlock()
	}
}

func (p *process) append(r *request) {
	p.requestQueue = append(p.requestQueue, r)
}

func (p *process) delete() {
	last := len(p.requestQueue) - 1
	p.requestQueue[0], p.requestQueue[last] = p.requestQueue[last], p.requestQueue[0]
	p.requestQueue = p.requestQueue[:last]
}

func (p *process) updateMinReceiveTime() {
	idx := (p.me + 1) % len(p.chans)
	minTime := p.receiveTime[idx]
	for i, t := range p.receiveTime {
		if i == p.me {
			continue
		}
		minTime = min(minTime, t)
	}
	p.minReceiveTime = minTime
}

func (p *process) occupyLoop() {
	for {
		<-p.toCheckRule5Chan

		p.rwmu.Lock()

		if len(p.requestQueue) > 0 && // p.requestQueue 中还有元素
			p.requestQueue[0].process == p.me && // 排在首位的 repuest 是 p 自己的
			p.requestQueue[0].time < p.minReceiveTime { // p 在 request 后，收到过所有其他 p 的回复

			p.occupy()

			// TODO: 这里需要 tick 一下吗
			p.clock.tick()
		}

		p.rwmu.Unlock()
	}
}
