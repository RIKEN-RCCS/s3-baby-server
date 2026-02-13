package server

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func monitor_enter(m *Monitor, rid uint64, duration int64, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("task%d to enter\n", rid)
	var ok, elapse = m.Enter("resource1", rid, (10 * time.Second))
	if ok {
		fmt.Printf("task%d entered (elapse=%v)\n", rid, elapse)
		time.Sleep(time.Duration(duration) * time.Second)
		m.Exit("resource1", rid)
	} else {
		fmt.Printf("task%d timeout (elapse=%v)\n", rid, elapse)
	}
}

func TestMonitorExclusion(t *testing.T) {
	fmt.Printf("Test Monitor Exclusion...\n")
	var wg sync.WaitGroup
	var m = New_monitor()
	go m.guard_loop()
	wg.Add(6)
	go monitor_enter(m, 101, 1, &wg)
	go monitor_enter(m, 102, 1, &wg)
	go monitor_enter(m, 103, 1, &wg)
	go monitor_enter(m, 104, 1, &wg)
	go monitor_enter(m, 105, 1, &wg)
	go monitor_enter(m, 106, 1, &wg)
	wg.Wait()
	close(m.schedule)
	time.Sleep(1 * time.Second)
	fmt.Printf("DONE\n")
}

func TestMonitorTimeout(t *testing.T) {
	fmt.Printf("Test Monitor Timeout...\n")
	var wg sync.WaitGroup
	var m = New_monitor()
	go m.guard_loop()
	wg.Add(6)
	go monitor_enter(m, 101, 30, &wg)
	time.Sleep(1 * time.Second)
	go monitor_enter(m, 102, 1, &wg)
	go monitor_enter(m, 103, 1, &wg)
	go monitor_enter(m, 104, 1, &wg)
	go monitor_enter(m, 105, 1, &wg)
	go monitor_enter(m, 106, 1, &wg)
	wg.Wait()
	close(m.schedule)
	time.Sleep(1 * time.Second)
	fmt.Printf("DONE\n")
}
