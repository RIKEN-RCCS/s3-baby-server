// monitor.go
// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

// A monitor to exclude accesses to the same object.  Entring fails by
// a timeout.  An entered task is allowed to run for the same duration
// of the timeout.  It forces to remove a task when it is slow.

package server

import (
	"fmt"
	"time"
	"sync"
)

type entered_task struct {
	id int64
	due time.Time
}

type monitor struct {
	resource map[string]entered_task
	mutex sync.Mutex
	blocker *sync.Cond
	blocked int
	timeout *time.Timer
	schedule chan struct{}
	trace bool
}

func (m *monitor) init() {
	m.resource = make(map[string]entered_task)
	m.blocker = sync.NewCond(&m.mutex)
	m.timeout = time.NewTimer(10 * time.Second)
	m.schedule = make(chan struct{})
	m.trace = false
}

// Calls broadcast to waiting tasks.  It loops forever.  It removes
// records of tasks which never or lately exit.  The loop exits when
// m.schedule is closed.
func (m *monitor) guard_loop() {
	for {
		var now = time.Now()
		var nextdue = now.Add(3600 * time.Second)
		m.mutex.Lock()
		for k, v := range m.resource {
			if v.due.Before(now) {
				delete(m.resource, k)
				if m.trace {
					fmt.Printf("monitor: delete slow task %#v\n", v.id)
				}
			}
			if v.due.Before(nextdue) {
				nextdue = v.due
			}
		}
		m.mutex.Unlock()
		var d = nextdue.Sub(now)
		if d > 0 {
			m.timeout.Reset(d)
			var ok bool
			select {
			case <- m.timeout.C:
				m.timeout.Stop()
			case _, ok = <- m.schedule:
				m.timeout.Stop()
				if !ok {
					return
				}
			}
		}
		if m.trace {
			fmt.Printf("monitor: wakeup!\n")
		}
		m.blocker.Broadcast()
	}
}

// Enters an exclusion region.  It returns false when timeout.  A
// failed task should not call m.exit().  It schedules the timer for a
// slow task.
func (m *monitor) enter(object string, id int64, d time.Duration) bool {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
		m.schedule <- struct{}{}
	}()
	var due = time.Now().Add(d)
	for true {
		if !time.Now().Before(due) {
			// A task fails to enter.
			return false
		}
		var _, inuse = m.resource[object]
		if !inuse {
			// A task enters.
			var rundue = time.Now().Add(d)
			m.resource[object] = entered_task{id, rundue}
			return true
		} else {
			m.blocked++
			m.blocker.Wait()
			m.blocked--
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
	var r = m.resource[object]
	if r.id == id {
		delete(m.resource, object)
	}
}
