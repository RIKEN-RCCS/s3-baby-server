package server

import (
    "testing"
    "fmt"
    "time"
	"sync"
)

func enter(m *monitor, id int64, duration int64, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("task%d to enter\n", id)
	var ok = m.enter("resource1", id, (10 * time.Second))
	if ok {
		fmt.Printf("task%d entered\n", id)
		time.Sleep(time.Duration(duration) * time.Second)
		m.exit("resource1", id)
	} else {
		fmt.Printf("task%d timeout\n", id)
	}
}

func TestMonitorExclusion(t *testing.T) {
	fmt.Printf("Test Monitor Exclusion...\n")
	var wg sync.WaitGroup
	var m = monitor{}
	m.init()
	go m.guard_loop()
	wg.Add(6)
	go enter(&m, 101, 1, &wg)
	go enter(&m, 102, 1, &wg)
	go enter(&m, 103, 1, &wg)
	go enter(&m, 104, 1, &wg)
	go enter(&m, 105, 1, &wg)
	go enter(&m, 106, 1, &wg)
	wg.Wait()
	close(m.schedule)
	time.Sleep(1 * time.Second)
	fmt.Printf("DONE\n")
}

func TestMonitorTimeout(t *testing.T) {
	fmt.Printf("Test Monitor Timeout...\n")
	var wg sync.WaitGroup
	var m = monitor{}
	m.init()
	go m.guard_loop()
	wg.Add(6)
	go enter(&m, 101, 30, &wg)
	time.Sleep(1 * time.Second)
	go enter(&m, 102, 1, &wg)
	go enter(&m, 103, 1, &wg)
	go enter(&m, 104, 1, &wg)
	go enter(&m, 105, 1, &wg)
	go enter(&m, 106, 1, &wg)
	wg.Wait()
	close(m.schedule)
	time.Sleep(1 * time.Second)
	fmt.Printf("DONE\n")
}
