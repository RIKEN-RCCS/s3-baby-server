// monitor.go

// Copyright 2025-2026 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// A Monitor Implementation to Serialize Operations

// A monitor is used to serialize accesses to the same object.  It
// services in fifo order.  Entering may fail by a timeout.

// NOTE: It takes a short sleep, when some tasks timeout.  It is to
// give tasks a time to leave themselves from the wait queue.  Without
// a sleep, worthless signals are delivered to a condition variable.
// Parameter m.smallwait controls it.  It should be a fraction of
// typical timeouts.
//
// NOTE: Make sure sending to a channel (m.schedule) be outside of a
// mutex.

package server

import (
	"fmt"
	"log"
	"slices"
	"sync"
	"time"
)

type Monitor struct {
	waitings  map[string][]wait_task
	blocker   *sync.Cond
	timer     *time.Timer
	schedule  chan struct{}
	mutex     sync.Mutex
	smallwait time.Duration
	trace     bool
}

type wait_task struct {
	rid   uint64
	due   time.Time
	start time.Time
}

func New_monitor() *Monitor {
	var m = Monitor{}
	m.init()
	return &m
}

func (m *Monitor) init() {
	m.waitings = make(map[string][]wait_task)
	m.blocker = sync.NewCond(&m.mutex)
	m.timer = time.NewTimer(10 * time.Second)
	m.schedule = make(chan struct{})
	m.smallwait = (1 * time.Millisecond)
	m.trace = false
}

// GUARD_LOOP broadcasts events to waiting tasks.  The loop runs
// forever, until m.schedule is closed.  Note the first entry in the
// queues is in service.
func (m *Monitor) Guard_loop() {
	for {
		var now = time.Now()
		var nextdue = now.Add(3600 * time.Second)
		m.mutex.Lock()
		for _, q := range m.waitings {
			if len(q) >= 2 {
				for _, e := range q[1:] {
					if e.due.Before(nextdue) {
						nextdue = e.due
					}
				}
			}
		}
		m.mutex.Unlock()
		var d = max(nextdue.Sub(now), m.smallwait)
		if m.trace {
			fmt.Printf("monitor: sleep %v\n", d)
		}
		m.timer.Reset(d)
		var ok bool
		select {
		case <-m.timer.C:
			m.timer.Stop()
		case _, ok = <-m.schedule:
			m.timer.Stop()
			if !ok {
				return
			}
		}
		if m.trace {
			fmt.Printf("monitor: wakeup\n")
		}
		m.blocker.Broadcast()
	}
}

// ENTER enters an exclusion region for an OBJECT.  RID be a unique
// key among requester.  It returns false when timeout.  A failed task
// should not call m.Exit().  It schedules the timer for a timeout.  A
// race of notifications and intervening deletions is acceptable.
func (m *Monitor) Enter(object string, rid uint64, d time.Duration) (bool, time.Duration) {
	var enter_time = time.Now()
	var due = enter_time.Add(d)
	func() {
		m.mutex.Lock()
		defer func() {
			m.mutex.Unlock()
			m.schedule <- struct{}{}
		}()
		var queue1 = m.waitings[object]
		m.waitings[object] = append(queue1, wait_task{rid, due, enter_time})
	}()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for {
		var queue2 = m.waitings[object]
		if len(queue2) == 0 {
			// Queue must has itself at least.
			log.Fatal("monitor: BAD queue state at enter")
		}
		if !time.Now().Before(due) {
			// A task fails to enter.
			var elapse = time.Since(enter_time)
			var i = slices.IndexFunc(queue2, func(e wait_task) bool {
				return (e.rid == rid)
			})
			if i == -1 {
				log.Fatal("monitor: BAD queue state at timeout")
			}
			m.waitings[object] = slices.Delete(queue2, i, i+1)
			if len(m.waitings[object]) == 0 {
				delete(m.waitings, object)
			}
			return false, elapse
		} else if queue2[0].rid == rid {
			// A task enters.
			queue2[0].start = time.Now()
			var elapse = time.Since(enter_time)
			return true, elapse
		} else {
			m.blocker.Wait()
		}
	}
	log.Print("monitor: BAD exit erroneously")
	var elapse = time.Since(enter_time)
	return false, elapse
}

// EXIT exits an exclusion region.  It schedules a next task.  Timeout
// tasks should not call Exit.  It shrinks m.waitings not to
// accumulate garbage.  It returns the consumed time in exclusion.
func (m *Monitor) Exit(object string, rid uint64) time.Duration {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
		m.schedule <- struct{}{}
	}()
	var queue1 = m.waitings[object]
	if !(len(queue1) != 0 && queue1[0].rid == rid) {
		log.Fatal("monitor: BAD queue state at exit")
	}
	var elapse = time.Since(queue1[0].start)
	m.waitings[object] = slices.Delete(queue1, 0, 1)
	if len(m.waitings[object]) == 0 {
		delete(m.waitings, object)
	}
	return elapse
}

// ATTEST returns a caller is in a monitor for an object.
func (m *Monitor) Attest(object string, rid uint64) bool {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
	}()
	var queue1 = m.waitings[object]
	if len(queue1) != 0 && queue1[0].rid == rid {
		return true
	} else {
		return false
	}
}

// DONE ends the use of a monitor.  It stops the thread for timeout
// wakeup.
func (m *Monitor) Done() {
	close(m.schedule)
}
