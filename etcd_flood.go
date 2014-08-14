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
	readers     int
	watchers    int
	machines    []string
	stop        chan chan struct{}
	running     bool
}

type Result struct {
	success bool
	dt      time.Duration
}

func NewETCDFlood(storeSize int, concurrency int, readers int, watchers int, machines []string) *ETCDFlood {
	return &ETCDFlood{
		storeSize:   storeSize,
		concurrency: concurrency,
		readers:     readers,
		watchers:    watchers,
		machines:    machines,
		stop:        make(chan chan struct{}, 0),
	}
}

func (f *ETCDFlood) Flood() {
	f.running = true

	wg := &sync.WaitGroup{}
	wg.Add(f.concurrency + f.readers + f.watchers)

	stop := make(chan struct{})
	inputStream := make(chan int)

	floodResultChan := make(chan Result)
	readResultChan := make(chan Result)
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
		floodResults := []Result{}
		readResults := []Result{}

		nSuccessfulFloods := 0
		nAttemptedFloods := 0
		lastWriteCount := 0

		nSuccesfulReads := 0
		nAttemptedReads := 0
		lastReadCount := 0

		t := time.Now()
		for {
			select {
			case result := <-floodResultChan:
				if result.success {
					nSuccessfulFloods += 1
				}
				nAttemptedFloods += 1
				if nAttemptedFloods%1000 == 0 {
					delta := nSuccessfulFloods - lastWriteCount
					update := fmt.Sprintf("WROTE %d/%d in %s (∆=%d/1000)", nSuccessfulFloods, nAttemptedFloods, time.Since(t), nSuccessfulFloods-lastWriteCount)
					if delta == 1000 {
						fmt.Println(greenColor + update + defaultStyle)
					} else {
						fmt.Println(redColor + update + defaultStyle)
					}
					lastWriteCount = nSuccessfulFloods
				}
				floodResults = append(floodResults, result)
			case result := <-readResultChan:
				if result.success {
					nSuccesfulReads += 1
				}
				nAttemptedReads += 1
				if nAttemptedReads%100 == 0 {
					delta := nSuccesfulReads - lastReadCount
					update := fmt.Sprintf("READ %d/%d in %s (∆=%d/100)", nSuccesfulReads, nAttemptedReads, time.Since(t), nSuccesfulReads-lastReadCount)
					if delta == 100 {
						fmt.Println(greenColor + update + defaultStyle)
					} else {
						fmt.Println(redColor + update + defaultStyle)
					}
					lastReadCount = nSuccesfulReads
				}
				readResults = append(readResults, result)
			case reply := <-printReport:
				f.printReport(floodResults, readResults, time.Since(t))
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
					floodResultChan <- Result{
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

	//readers
	for i := 0; i < f.readers; i++ {
		go func() {
			client := etcd.NewClient(f.machines)
			ticker := time.NewTicker(10 * time.Millisecond)
			for {
				select {
				case <-ticker.C:
					t := time.Now()
					_, err := client.Get("/flood", false, true)
					readResultChan <- Result{
						success: err == nil,
						dt:      time.Since(t),
					}
				case <-stop:
					ticker.Stop()
					wg.Done()
					return
				}
			}
		}()
	}

}

func (f *ETCDFlood) printReport(floodResults []Result, readResults []Result, dt time.Duration) {
	f.printSubReport("Write", "Wrote", "writes", "write", floodResults, dt)
	slowest := time.Duration(0)
	fastest := time.Hour
	nSuccess := 0
	totalWriteTime := time.Duration(0)
	for _, result := range floodResults {
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

	nAttempts := len(floodResults)
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

func (f *ETCDFlood) printSubReport(floodResults []Result, readResults []Result, dt time.Duration) {
	//HERE!
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
