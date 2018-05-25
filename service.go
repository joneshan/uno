// Copyright 2018 acrazing <joking.young@gmail.com>. All rights reserved.
// Since 2018-05-24 09:06:59
// Version 1.0.0
// Desc The uuid service
package uno

import (
	"context"
	"time"
)

var empty = Empty{}

// the value will be used
type poolNode struct {
	value uint32
	next  *poolNode
}

// the value is using
// if expired is false, the node is living, and the expiresAt means it
// will be freezed at the time
// else the node is freezing, and the expiresAt means it will be freed
// at the time
type busyNode struct {
	expired   bool
	value     uint32
	expiresAt int64
	next      *busyNode
	prev      *busyNode
}

type rentCmd struct {
	result chan uint32
}

type reletCmd struct {
	value  uint32
	result chan bool
}

type Options struct {
	// auto allocate ticket pool poolSize, default: 10
	PoolVolume uint32
	// time to live, if not relet, after ttl it will be reused
	// this is used for avoid client breaking down without return
	// its tickets.
	TTL time.Duration
	// time to freeze after a ticket has been returned
	TTF time.Duration
	// the min value of the ticket, include itself
	MinValue uint32
	// the max value of the ticker, exclude itself, which means
	// the range of ticket is [MinValue, MaxValue)
	MaxValue uint32
}

type Worker struct {
	Options
	// signal for rent a new ticket
	RentSignal chan *rentCmd
	// signal for relet a rent ticket
	ReletSignal chan *reletCmd
	// signal for return a ticket, this will not return any value
	ReturnSignal chan uint32
	// the nextValue rentable value in pool
	poolHead *poolNode
	// the poolTail rent value in pool
	poolTail *poolNode
	// the values poolSize in pool
	poolSize uint32
	// the nextValue value to be allocated for pool
	nextValue uint32
	// the rented values map, use for check to create pool
	busyNodes map[uint32]*busyNode
	// the first one node is freezing
	freezeHead *busyNode
	// the last one is freezing
	freezeTail *busyNode
	// the first one is rented
	rentHead *busyNode
	// the last one is rented
	rentTail *busyNode
	// we must use two timer and two chain to avoid insert
	// node to the chain, if TTL != TTF
	// TTL/TTF is constant, rent/relet/expire event just need
	// to move the node to the tail of the chain
	freeTimer   *time.Timer
	expireTimer *time.Timer
	busySize    uint32
}

func NewWorker() *Worker {
	return &Worker{
		Options: Options{
			PoolVolume: 100,
			TTL:        30 * time.Minute,
			TTF:        30 * time.Minute,
			MinValue:   1e5,
			MaxValue:   2e5,
		},
		RentSignal:   make(chan *rentCmd),
		ReletSignal:  make(chan *reletCmd),
		ReturnSignal: make(chan uint32),
		nextValue:    1e5,
		busyNodes:    map[uint32]*busyNode{},
	}
}

// allocate available id list to pool
func (w *Worker) allocate() {
	var curr *poolNode
	var i uint32
	if w.busySize >= w.MaxValue-w.MinValue {
		return
	}
	var size = w.MaxValue - w.MinValue - w.busySize
	if size > w.PoolVolume {
		size = w.PoolVolume
	}
	for ; i < size; i++ {
		for {
			if _, ok := w.busyNodes[w.nextValue]; ok {
				w.nextValue += 1
				if w.nextValue >= w.MaxValue {
					w.nextValue = w.MinValue
				}
				continue
			} else {
				break
			}
		}
		if curr == nil {
			curr = &poolNode{value: w.nextValue}
			w.poolHead = curr
		} else {
			curr.next = &poolNode{value: w.nextValue}
			curr = curr.next
		}
		w.nextValue += 1
		if w.nextValue == w.MaxValue {
			w.nextValue = w.MinValue
		}
	}
	w.poolTail = curr
	w.poolSize = size
}

// reset the expire timer
func (w *Worker) resetExpireTimer() {
	if w.rentHead == nil {
		w.expireTimer.Reset(w.TTL)
	} else {
		w.expireTimer.Reset(time.Duration(w.rentHead.expiresAt - time.Now().UnixNano()))
	}
}

// reset the free timer
func (w *Worker) resetFreeTimer() {
	if w.freezeHead == nil {
		w.freeTimer.Reset(w.TTF)
	} else {
		w.freeTimer.Reset(time.Duration(w.freezeHead.expiresAt - time.Now().UnixNano()))
	}
}

// rent a value, start expire timer
func (w *Worker) rent() uint32 {
	// log.Printf("worker.rent busy size: %d, pool size: %d, pool head: %v, rent head: %v, freeze head: %v", w.busySize, w.poolSize, w.poolHead, w.rentHead, w.freezeHead)
	if w.poolSize == 0 {
		w.allocate()
	}
	if w.poolSize == 0 {
		return 0
	}
	first := w.poolHead
	node := busyNode{false, first.value, time.Now().Add(w.TTL).UnixNano(), nil, w.rentTail}
	w.busyNodes[first.value] = &node
	if w.rentTail == nil {
		w.rentHead = &node
		w.rentTail = &node
		w.resetExpireTimer()
	} else {
		w.rentTail.next = &node
		w.rentTail = &node
	}
	w.poolHead = first.next
	if first.next == nil {
		w.poolTail = nil
	} else {
		first.next = nil
	}
	w.poolSize -= 1
	w.busySize += 1
	return first.value
}

// relet a value, reset expire timer
func (w *Worker) relet(value uint32) bool {
	node, ok := w.busyNodes[value]
	// log.Printf("worker.relet: %d, node: %v, busy size: %d, pool size: %d, pool head: %v, rent head: %v, freeze head: %v", value, node, w.busySize, w.poolSize, w.poolHead, w.rentHead, w.freezeHead)
	if !ok || node.expired {
		return false
	}
	node.expiresAt = time.Now().Add(w.TTL).UnixNano()
	var isHead = node.prev == nil
	var isTail = node.next == nil
	if isHead {
		if !isTail {
			// if only one node, will not update the chain
			// else will update head, and move it to tile
			w.rentHead = node.next
		}
		// is head one, must update the timer
		w.resetExpireTimer()
		if isTail {
			// only one node will not update the chain
			return true
		}
	}
	if !isTail {
		if !isHead {
			node.prev.next = node.next
		}
		node.next.prev = node.prev
		node.next = nil
		node.prev = w.rentTail
		w.rentTail.next = node
		w.rentTail = w.rentTail.next
	}
	return true
}

// freeze a value, freeze it
func (w *Worker) freeze(value uint32) {
	node, ok := w.busyNodes[value]
	// log.Printf("worker.freeze: %d, node: %v, busy size: %d, pool size: %d, pool head: %v, rent head: %v, freeze head: %v", value, node, w.busySize, w.poolSize, w.poolHead, w.rentHead, w.freezeHead)
	if !ok || node.expired {
		return
	}
	// drop it from rent chain at first
	if node.prev == nil {
		w.rentHead = node.next
		w.resetExpireTimer()
	} else {
		node.prev.next = node.next
	}
	if node.next == nil {
		w.rentTail = node.prev
	} else {
		node.next.prev = node.prev
	}
	node.prev = nil
	node.next = nil
	// append to freeze chain
	node.expired = true
	node.expiresAt = time.Now().Add(w.TTF).UnixNano()
	if w.freezeHead == nil {
		w.freezeHead = node
		w.freezeTail = node
		w.resetFreeTimer()
	} else {
		w.freezeTail.next = node
		node.prev = w.freezeTail
		w.freezeTail = node
	}
}

// rent expire, freeze it
func (w *Worker) expire() {
	// log.Printf("worker.expire, busy size: %d, pool size: %d, pool head: %v, rent head: %v, freeze head: %v", w.busySize, w.poolSize, w.poolHead, w.rentHead, w.freezeHead)
	if w.rentHead == nil {
		w.resetExpireTimer()
		return
	}
	// auto freeze the expired value
	w.freeze(w.rentHead.value)
}

// free a value
func (w *Worker) free() {
	// log.Printf("worker.freeze, busy size: %d, pool size: %d, pool head: %v, rent head: %v, freeze head: %v", w.busySize, w.poolSize, w.poolHead, w.rentHead, w.freezeHead)
	if w.freezeHead == nil {
		w.resetFreeTimer()
		return
	}
	// drop freeze head
	node := w.freezeHead
	w.freezeHead = node.next
	if node.next != nil {
		node.next.prev = nil
	}
	node.next = nil
	delete(w.busyNodes, node.value)
	w.resetFreeTimer()
	w.busySize -= 1
	// append to pool chain
	if w.poolSize > 5*w.PoolVolume {
		// too much nodes on pool chain, skip it
		return
	}
	pool := &poolNode{node.value, nil}
	if w.poolHead == nil {
		w.poolHead = pool
		w.poolTail = pool
	} else {
		w.poolTail.next = pool
		w.poolTail = pool
	}
	w.poolSize += 1
}

// close worker
func (w *Worker) close() {
	close(w.RentSignal)
	close(w.ReletSignal)
	close(w.ReturnSignal)
	w.expireTimer.Stop()
	w.freeTimer.Stop()
}

func (w *Worker) Init(opt *Options) {
	if opt.PoolVolume > 0 {
		w.PoolVolume = opt.PoolVolume
	}
	if opt.TTL > 0 {
		w.TTL = opt.TTL
	}
	if opt.TTF > 0 {
		w.TTF = opt.TTF
	}
	if opt.MinValue > 0 {
		w.MinValue = opt.MinValue
		w.nextValue = opt.MinValue
	}
	if opt.MaxValue > 0 {
		w.MaxValue = opt.MaxValue
	}
	if opt.PoolVolume > w.MaxValue-w.MinValue {
		opt.PoolVolume = w.MinValue - w.MinValue
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.expireTimer = time.NewTimer(w.TTL)
	w.freeTimer = time.NewTimer(w.TTF)
	w.resetExpireTimer()
	w.resetFreeTimer()
	for {
		select {
		case <-ctx.Done():
			w.close()
			return
		case cmd, ok := <-w.RentSignal:
			if !ok {
				return
			}
			cmd.result <- w.rent()
		case cmd, ok := <-w.ReletSignal:
			if !ok {
				return
			}
			cmd.result <- w.relet(cmd.value)
		case value, ok := <-w.ReturnSignal:
			if !ok {
				return
			}
			w.freeze(value)
		case _, ok := <-w.expireTimer.C:
			if !ok {
				return
			}
			w.expire()
		case _, ok := <-w.freeTimer.C:
			if !ok {
				return
			}
			w.free()
		}
	}
}

func (w *Worker) Dump() []byte {
	// TODO
	return nil
}

func (w *Worker) Load(status []byte) {
	// TODO
}

func (w *Worker) Rent() uint32 {
	cmd := &rentCmd{make(chan uint32)}
	w.RentSignal <- cmd
	return <-cmd.result
}

func (w *Worker) Relet(value uint32) bool {
	cmd := &reletCmd{value, make(chan bool)}
	w.ReletSignal <- cmd
	return <-cmd.result
}

func (w *Worker) Return(value uint32) {
	w.ReturnSignal <- value
}
