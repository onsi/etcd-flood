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
	storeSize    int
	concurrency  int
	heavyReaders int
	lightReaders int
	watchers     int
	machines     []string
	stop         chan chan struct{}
	running      bool
}

type Result struct {
	success bool
	dt      time.Duration
}

type ResultAggregator struct {
	NSuccesful, NAttempted, LastCount                   int
	ReportName, PastTenseVerb, SingularNoun, PluralNoun string
	Results                                             []Result
	StartTime                                           time.Time
	UpdateThreshold                                     int
}

func (r *ResultAggregator) AppendResult(result Result) {
	if result.success {
		r.NSuccesful += 1
	}
	r.NAttempted += 1
	if r.NAttempted%r.UpdateThreshold == 0 {
		delta := r.NSuccesful - r.LastCount
		update := fmt.Sprintf("%s %d/%d in %s (∆=%d/%d)", r.PastTenseVerb, r.NSuccesful, r.NAttempted, time.Since(r.StartTime), r.NSuccesful-r.LastCount, r.UpdateThreshold)
		if delta == r.UpdateThreshold {
			fmt.Println(greenColor + update + defaultStyle)
		} else {
			fmt.Println(redColor + update + defaultStyle)
		}
		r.LastCount = r.NSuccesful
	}
	r.Results = append(r.Results, result)
}

func (r ResultAggregator) PrintReport() {
	if r.NAttempted == 0 {
		return
	}

	dt := time.Since(r.StartTime)
	slowest := time.Duration(0)
	fastest := time.Hour
	totalTime := time.Duration(0)
	for _, result := range r.Results {
		if result.dt > slowest {
			slowest = result.dt
		}
		if result.dt < fastest {
			fastest = result.dt
		}
		totalTime += result.dt
	}

	numFailed := r.NAttempted - r.NSuccesful
	fractionFailed := float64(numFailed) / float64(r.NAttempted) * 100.0

	CyanBanner(r.ReportName)
	fmt.Println("")
	fmt.Printf(cyanColor+"  %s %d/%d (missed %d = %.2f%%) in %s\n"+defaultStyle, r.PastTenseVerb, r.NSuccesful, r.NAttempted, numFailed, fractionFailed, dt)
	fmt.Printf(cyanColor+"  That's %.2f succesful %s per second\n"+defaultStyle, float64(r.NSuccesful)/dt.Seconds(), r.PluralNoun)
	fmt.Printf(cyanColor+"  And    %.2f attempts per second\n"+defaultStyle, float64(r.NAttempted)/dt.Seconds())
	fmt.Printf(cyanColor+"  Fastest %s took %s\n"+defaultStyle, r.SingularNoun, fastest)
	fmt.Printf(cyanColor+"  Slowest %s took %s\n"+defaultStyle, r.SingularNoun, slowest)
	fmt.Printf(cyanColor+"  Mean %s took %s\n"+defaultStyle, r.SingularNoun, totalTime/time.Duration(r.NAttempted))
	fmt.Println("")
}

func NewETCDFlood(storeSize int, concurrency int, heavyReaders int, lightReaders int, watchers int, machines []string) *ETCDFlood {
	return &ETCDFlood{
		storeSize:    storeSize,
		concurrency:  concurrency,
		heavyReaders: heavyReaders,
		lightReaders: lightReaders,
		watchers:     watchers,
		machines:     machines,
		stop:         make(chan chan struct{}, 0),
	}
}

func (f *ETCDFlood) Flood() {
	f.running = true

	wg := &sync.WaitGroup{}
	wg.Add(f.concurrency + f.heavyReaders + f.lightReaders + f.watchers)

	stop := make(chan struct{})
	inputStream := make(chan int)

	floodResultChan := make(chan Result)
	heavyReadResultChan := make(chan Result)
	lightReadResultChan := make(chan Result)
	watchResultChan := make(chan Result)
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
		floodAggregator := &ResultAggregator{
			ReportName:      "Write Report",
			PastTenseVerb:   "Wrote",
			SingularNoun:    "write",
			PluralNoun:      "writes",
			StartTime:       time.Now(),
			UpdateThreshold: 1000,
		}

		heavyReadAggregator := &ResultAggregator{
			ReportName:      "Heavy Read Report",
			PastTenseVerb:   "(Heavy) Read",
			SingularNoun:    "heavy read",
			PluralNoun:      "heavy reads",
			StartTime:       time.Now(),
			UpdateThreshold: 100,
		}

		lightReadAggregator := &ResultAggregator{
			ReportName:      "Light Read Report",
			PastTenseVerb:   "(Light) Read",
			SingularNoun:    "light read",
			PluralNoun:      "light reads",
			StartTime:       time.Now(),
			UpdateThreshold: 1000,
		}

		watchAggregator := &ResultAggregator{
			ReportName:      "Watch Report",
			PastTenseVerb:   "Watch",
			SingularNoun:    "watch",
			PluralNoun:      "watches",
			StartTime:       time.Now(),
			UpdateThreshold: 1000,
		}

		for {
			select {
			case result := <-floodResultChan:
				floodAggregator.AppendResult(result)
			case result := <-heavyReadResultChan:
				heavyReadAggregator.AppendResult(result)
			case result := <-lightReadResultChan:
				lightReadAggregator.AppendResult(result)
			case result := <-watchResultChan:
				watchAggregator.AppendResult(result)
			case reply := <-printReport:
				floodAggregator.PrintReport()
				heavyReadAggregator.PrintReport()
				lightReadAggregator.PrintReport()
				watchAggregator.PrintReport()
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

	//heavy readers
	for i := 0; i < f.heavyReaders; i++ {
		go func() {
			client := etcd.NewClient(f.machines)
			client.SetConsistency(etcd.WEAK_CONSISTENCY)
			ticker := time.NewTicker(10 * time.Millisecond)
			for {
				select {
				case <-ticker.C:
					t := time.Now()
					_, err := client.Get("/flood", false, true)
					heavyReadResultChan <- Result{
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

	//light readers
	storeSize := f.storeSize
	for i := 0; i < f.lightReaders; i++ {
		go func() {
			client := etcd.NewClient(f.machines)
			client.SetConsistency(etcd.WEAK_CONSISTENCY)
			ticker := time.NewTicker(10 * time.Millisecond)
			key := 0
			for {
				select {
				case <-ticker.C:
					t := time.Now()
					_, err := client.Get(fmt.Sprintf("/flood/key-%d", key%(storeSize/100)), false, true)
					key++
					success := err == nil
					if !success {
						etcdErr, ok := err.(*etcd.EtcdError)
						if ok {
							success = etcdErr.ErrorCode == 100
						}
					}
					lightReadResultChan <- Result{
						success: success,
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

	//watchers
	for i := 0; i < f.watchers; i++ {
		go func() {
			client := etcd.NewClient(f.machines)
			var stopChan = make(chan bool)
			var receiver = make(chan *etcd.Response)
			client.Watch("/flood", 0, true, receiver, stopChan)
			t := time.Now()
			for {
				select {
				case _, ok := <-receiver:
					watchResultChan <- Result{
						success: ok,
						dt:      time.Since(t),
					}
					t = time.Now()
					if !ok {
						stopChan = make(chan bool)
						receiver = make(chan *etcd.Response)
						client.Watch("/flood", 0, true, receiver, stopChan)
					}
				case <-stop:
					stopChan <- true
					wg.Done()
					return
				}
			}
		}()
	}
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
