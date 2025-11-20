// monitor.go
// Copyright 2025-2025 RIKEN R-CCS
// SPDX-License-Identifier: BSD-2-Clause

// A monitor to exclude accesses to the same object.  It serves in
// fifo order.  Entering may fail by a timeout.

// It takes a short sleep, when some tasks timeout.  It is to give
// them a time to leave from the wait queue.  Without a sleep, it
// makes worthless signals to a condition variable.

// NOTE: Make sure sending to a channel be outside of a mutex.

package server

import (
	"fmt"
	"log"
	"time"
	"slices"
	"sync"
)

type monitor struct {
	waitings map[string][]wait_task
	blocker *sync.Cond
	timer *time.Timer
	schedule chan struct{}
	mutex sync.Mutex
	trace bool
}

type wait_task struct {
	id int64
	due time.Time
}

func new_monitor() *monitor {
	var m = monitor{}
	m.init()
	return &m
}

func (m *monitor) init() {
	m.waitings = make(map[string][]wait_task)
	m.blocker = sync.NewCond(&m.mutex)
	m.timer = time.NewTimer(10 * time.Second)
	m.schedule = make(chan struct{})
	m.trace = false
}

// Broadcasts events to waiting tasks.  The loop runs forever, until
// m.schedule is closed.  Note first queue entries are not waiting.
func (m *monitor) guard_loop() {
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
		var d = nextdue.Sub(now) + (1 * time.Millisecond)
		if d <= 0 {
			// Take small sleep when some tasks timeout.
			d = 1 * time.Millisecond
		}
		if m.trace {fmt.Printf("monitor: sleep %v\n", d)}
		m.timer.Reset(d)
		var ok bool
		select {
		case <- m.timer.C:
			m.timer.Stop()
		case _, ok = <- m.schedule:
			m.timer.Stop()
			if !ok {
				return
			}
		}
		if m.trace {fmt.Printf("monitor: wakeup\n")}
		m.blocker.Broadcast()
	}
}

// Enters an exclusion region.  It returns false when timeout.  A
// failed task should not call m.exit().  It schedules the timer for a
// timeout.  A race of notifications and intervening deletions is
// acceptable.  Deletions are OK.
func (m *monitor) enter(object string, id int64, d time.Duration) bool {
	var due = time.Now().Add(d)
	func () {
		m.mutex.Lock()
		defer func() {
			m.mutex.Unlock()
			m.schedule <- struct{}{}
		}()
		var queue1 = m.waitings[object]
		m.waitings[object] = append(queue1, wait_task{id, due})
	}()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for true {
		var queue2 = m.waitings[object]
		if (len(queue2) == 0) {
			// Itself exists at least.
			log.Fatal("monitor: BAD queue state at enter")
		}
		if !time.Now().Before(due) {
			// A task fails to enter.
			var i = slices.IndexFunc(queue2, func(e wait_task) bool {
				return e.id == id})
			if (i == -1) {
				log.Fatal("monitor: BAD queue state at timeout")
			}
			m.waitings[object] = slices.Delete(queue2, i, i + 1)
			if len(m.waitings[object]) == 0 {
				delete(m.waitings, object)
			}
			return false
		} else if queue2[0].id == id {
			// A task enters.
			return true
		} else {
			m.blocker.Wait()
		}
	}
	return false
}

// Exits an exclusion region.  It schedules for a next task.
func (m *monitor) exit(object string, id int64) {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
		m.schedule <- struct{}{}
	}()
	var queue1 = m.waitings[object]
	if !(len(queue1) != 0 && queue1[0].id == id) {
		log.Fatal("monitor: BAD queue state at exit")
	}
	m.waitings[object] = slices.Delete(queue1, 0, 1)
	if len(m.waitings[object]) == 0 {
		delete(m.waitings, object)
	}
}
