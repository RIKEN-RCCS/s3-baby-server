// monitor.go

// A monitor to exclude accesses to the same object.  It has timeout.
// It wake-ups all waiting threads, which check their exclusion.

package server

import (
	"time"
	"sync"
)

type monitor struct {
	resource map[string]time.Time
	mutex sync.Mutex
	blocker *sync.Cond
	timeout time.Timer
	reschedule chan struct{}
}

func (m *monitor) start_wakeup_timer() {
	m.blocker = sync.NewCond(&m.mutex)
	m.timeout = *time.NewTimer(0 * time.Second)
	for true {
		var now = time.Now()
		m.mutex.Lock()
		var next = now.Add(10 * time.Second)
		for _, v := range m.resource {
			if v.Before(next) {
				next = v
			}
		}
		defer m.mutex.Unlock()
		var d time.Duration = next.Sub(now)
		m.timeout.Reset(d)
		select {
        case <- m.timeout.C:
            m.timeout.Stop()
        case <- m.reschedule:
            m.timeout.Stop()
        }
		m.blocker.Broadcast()
	}
}

// It returns false when timeout.
func (m *monitor) enter(object string, d time.Duration) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var duetime = time.Now().Add(d)
	for true {
		if !time.Now().Before(duetime) {
			return false
		}
		var _, inuse = m.resource[object]
		if !inuse {
			m.resource[object] = duetime
			return true
		} else {
			m.blocker.Wait()
		}
	}
	return false
}

// It re-schedules the timer for a next timeout, which wake-ups all
// waiting threads.
func (m *monitor) exit(object string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.resource, object)
	m.reschedule <- struct{}{}
}
