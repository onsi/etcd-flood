package flood

import (
	"fmt"
	"sync"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

type Flood struct {
	storeSize    int
	writers      int
	heavyReaders int
	lightReaders int
	watchers     int
	machines     []string
	stop         chan chan struct{}
	running      bool
}

func NewFlood(storeSize int, writers int, heavyReaders int, lightReaders int, watchers int, machines []string) *Flood {
	return &Flood{
		storeSize:    storeSize,
		writers:      writers,
		heavyReaders: heavyReaders,
		lightReaders: lightReaders,
		watchers:     watchers,
		machines:     machines,
		stop:         make(chan chan struct{}, 0),
	}
}

func (f *Flood) Flood() {
	f.running = true

	wg := &sync.WaitGroup{}
	wg.Add(f.writers + f.heavyReaders + f.lightReaders + f.watchers)

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
			LastTime:        time.Now(),
			UpdateThreshold: 1000,
		}

		heavyReadAggregator := &ResultAggregator{
			ReportName:      "Heavy Read Report",
			PastTenseVerb:   "(Heavy) Read",
			SingularNoun:    "heavy read",
			PluralNoun:      "heavy reads",
			StartTime:       time.Now(),
			LastTime:        time.Now(),
			UpdateThreshold: 20,
		}

		lightReadAggregator := &ResultAggregator{
			ReportName:      "Light Read Report",
			PastTenseVerb:   "(Light) Read",
			SingularNoun:    "light read",
			PluralNoun:      "light reads",
			StartTime:       time.Now(),
			LastTime:        time.Now(),
			UpdateThreshold: 1000,
		}

		watchAggregator := &ResultAggregator{
			ReportName:      "Watch Report",
			PastTenseVerb:   "Watch",
			SingularNoun:    "watch",
			PluralNoun:      "watches",
			StartTime:       time.Now(),
			LastTime:        time.Now(),
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
	for i := 0; i < f.writers; i++ {
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
					_, err := client.Get(fmt.Sprintf("/flood/key-%d", key%(storeSize)), false, true)
					key++
					success := err == nil
					if !success {
						success = err.(*etcd.EtcdError).ErrorCode == 100
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

func (f *Flood) Stop() {
	if f.running {
		fmt.Println(redColor + "Stopping flood..." + defaultStyle)
		reply := make(chan struct{}, 0)
		f.stop <- reply
		<-reply
		fmt.Println(redColor + "...stopped" + defaultStyle)
	}
	f.running = false
}
