package etcd_flood

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

const defaultStyle = "\x1b[0m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"

type ETCDFlood struct {
	messagesPerTick int
	client          *etcd.Client
	semaphore       chan struct{}
	stop            chan chan struct{}
	running         bool
}

func NewETCDFlood(messagesPerTick int, concurrency int, machines []string) *ETCDFlood {
	return &ETCDFlood{
		messagesPerTick: messagesPerTick,
		client:          etcd.NewClient(machines),
		semaphore:       make(chan struct{}, concurrency),
		stop:            make(chan chan struct{}, 0),
	}
}

func (f *ETCDFlood) Flood() {
	ticker := time.NewTicker(100 * time.Millisecond)
	f.running = true
	go func() {
		for {
			select {
			case <-ticker.C:
				t := time.Now()

				fmt.Printf("Writing %d entries...\n", f.messagesPerTick)

				wg := &sync.WaitGroup{}
				wg.Add(f.messagesPerTick)
				numFailures := uint64(0)

				for i := 0; i < f.messagesPerTick; i++ {
					go func(i int) {
						f.semaphore <- struct{}{}
						_, err := f.client.Set(fmt.Sprintf("/flood/key-%d", i), "some-value", 0)
						if err != nil {
							atomic.AddUint64(&numFailures, 1)
						}
						<-f.semaphore
						wg.Done()
					}(i)
				}

				wg.Wait()

				nFailures := atomic.LoadUint64(&numFailures)
				if nFailures > 0 {
					RedBanner(fmt.Sprintf("Failed to write %d of %d keys%s", numFailures, f.messagesPerTick, defaultStyle))
				} else {
					GreenBanner(fmt.Sprintf("Wrote all %d keys!", f.messagesPerTick))
				}
				fmt.Printf("...done in %s\n", time.Since(t))
			case reply := <-f.stop:
				fmt.Println("Stopping flood...")
				ticker.Stop()
				reply <- struct{}{}
				fmt.Println("...stopped")
				return
			}
		}
	}()
}

func (f *ETCDFlood) Stop() {
	if f.running {
		reply := make(chan struct{}, 0)
		f.stop <- reply
		<-reply
	}
	f.running = false
}

func GreenBanner(s string) {
	banner(s, greenColor)
}

func RedBanner(s string) {
	banner(s, redColor)
}

func banner(s string, color string) {
	l := len(strings.Split(s, "\n")[0])
	fmt.Println("")
	fmt.Println(color + strings.Repeat("~", l) + defaultStyle)
	fmt.Println(color + s + defaultStyle)
	fmt.Println(color + strings.Repeat("~", l) + defaultStyle)
}
