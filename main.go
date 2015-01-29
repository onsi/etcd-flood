package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/onsi/etcd-flood/flood"
)

var STORE_SIZE int
var WRITERS int
var HEAVY_READERS int
var LIGHT_READERS int
var WATCHERS int
var MACHINES string

func init() {
	flag.IntVar(&STORE_SIZE, "storeSize", 30000, "total number of keys to put in the store")
	flag.IntVar(&WRITERS, "writers", 300, "number of concurrent write requests")
	flag.IntVar(&HEAVY_READERS, "heavyReaders", 5, "number of concurrent readers that fetch the entire store")
	flag.IntVar(&LIGHT_READERS, "lightReaders", 20, "number of concurrent readers that fetch a key at a time")
	flag.IntVar(&WATCHERS, "watchers", 0, "number of concurrent watchers")
	flag.StringVar(&MACHINES, "machines", "", "comma-separated list of etcd machines")
}

func main() {
	flag.Parse()
	etcdFlood := flood.NewFlood(STORE_SIZE, WRITERS, HEAVY_READERS, LIGHT_READERS, WATCHERS, strings.Split(MACHINES, ","))
	etcdFlood.Flood()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	<-c
	etcdFlood.Stop()
}
