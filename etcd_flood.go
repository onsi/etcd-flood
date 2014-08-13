package etcd_flood

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

const defaultStyle = "\x1b[0m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"
const yellowColor = "\x1b[33m"
const cyanColor = "\x1b[36m"

type ETCDFlood struct {
	storeSize   int
	concurrency int
	machines    []string
	stop        chan chan struct{}
	running     bool
}

type FloodResult struct {
	success bool
	dt      time.Duration
}

func NewETCDFlood(storeSize int, concurrency int, machines []string) *ETCDFlood {
	return &ETCDFlood{
		storeSize:   storeSize,
		concurrency: concurrency,
		machines:    machines,
		stop:        make(chan chan struct{}, 0),
	}
}

func (f *ETCDFlood) Flood() {
	f.running = true

	wg := &sync.WaitGroup{}
	wg.Add(f.concurrency)

	stop := make(chan struct{})
	inputStream := make(chan int)

	resultChan := make(chan FloodResult)
	printReport := make(chan chan struct{}, 0)

	//stopper
	go func() {
		reply := <-f.stop
		close(stop)
		wg.Wait()
		printReport <- reply
	}()

	//result aggregator
	go func() {
		results := []FloodResult{}
		nSuccess := 0
		nAttempted := 0
		lastCount := 0
		t := time.Now()
		for {
			select {
			case result := <-resultChan:
				if result.success {
					nSuccess += 1
				}
				nAttempted += 1
				if nAttempted%1000 == 0 {
					delta := nSuccess - lastCount
					update := fmt.Sprintf("Wrote %d/%d in %s (∆=%d/1000)", nSuccess, nAttempted, time.Since(t), nSuccess-lastCount)
					if delta == 1000 {
						fmt.Println(greenColor + update + defaultStyle)
					} else {
						fmt.Println(redColor + update + defaultStyle)
					}
					lastCount = nSuccess
				}
				results = append(results, result)
			case reply := <-printReport:
				f.printReport(results, time.Since(t))
				reply <- struct{}{}
				return
			}
		}
	}()

	//filler
	go func() {
		i := 0
		for {
			select {
			case inputStream <- (i % f.storeSize):
				i++
			case <-stop:
				return
			}
		}
	}()

	//flood
	for i := 0; i < f.concurrency; i++ {
		go func() {
			client := etcd.NewClient(f.machines)
			for {
				select {
				case key := <-inputStream:
					t := time.Now()
					_, err := client.Set(fmt.Sprintf("/flood/key-%d", key), fmt.Sprintf("some-value-%s", time.Now()), 0)
					resultChan <- FloodResult{
						success: err == nil,
						dt:      time.Since(t),
					}
				case <-stop:
					wg.Done()
					return
				}
			}
		}()
	}
}

func (f *ETCDFlood) printReport(results []FloodResult, dt time.Duration) {
	slowest := time.Duration(0)
	fastest := time.Hour
	nSuccess := 0
	totalWriteTime := time.Duration(0)
	for _, result := range results {
		if result.dt > slowest {
			slowest = result.dt
		}
		if result.dt < fastest {
			fastest = result.dt
		}
		if result.success {
			nSuccess++
		}
		totalWriteTime += result.dt
	}

	nAttempts := len(results)
	CyanBanner("Report")
	fmt.Println("")
	fmt.Printf(cyanColor+"  Wrote %d/%d (missed %d = %.2f%%) in %s\n"+defaultStyle, nSuccess, nAttempts, nAttempts-nSuccess, float64(nAttempts-nSuccess)/float64(nAttempts)*100.0, dt)
	fmt.Printf(cyanColor+"  That's %.2f succesful writes per second\n"+defaultStyle, float64(nSuccess)/dt.Seconds())
	fmt.Printf(cyanColor+"  And    %.2f attempts per second\n"+defaultStyle, float64(nAttempts)/dt.Seconds())
	fmt.Printf(cyanColor+"  Fastest write took %s\n"+defaultStyle, fastest)
	fmt.Printf(cyanColor+"  Slowest write took %s\n"+defaultStyle, slowest)
	fmt.Printf(cyanColor+"  Mean write took %s\n"+defaultStyle, totalWriteTime/time.Duration(nAttempts))
	fmt.Println("")
}

func (f *ETCDFlood) Stop() {
	if f.running {
		fmt.Println(redColor + "Stopping flood..." + defaultStyle)
		reply := make(chan struct{}, 0)
		f.stop <- reply
		<-reply
		fmt.Println(redColor + "...stopped" + defaultStyle)
	}
	f.running = false
}

func GreenBanner(s string) {
	banner(s, greenColor)
}

func CyanBanner(s string) {
	banner(s, cyanColor)
}

func RedBanner(s string) {
	banner(s, redColor)
}

func YellowBanner(s string) {
	banner(s, yellowColor)
}

func banner(s string, color string) {
	l := len(strings.Split(s, "\n")[0])
	fmt.Println("")
	fmt.Println(color + strings.Repeat("~", l) + defaultStyle)
	fmt.Println(color + s + defaultStyle)
	fmt.Println(color + strings.Repeat("~", l) + defaultStyle)
}
