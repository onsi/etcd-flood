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
	flag.IntVar(&WRITERS, "writers", 0, "number of concurrent write requests")
	flag.IntVar(&HEAVY_READERS, "heavyReaders", 0, "number of concurrent readers that fetch the entire store")
	flag.IntVar(&LIGHT_READERS, "lightReaders", 0, "number of concurrent readers that fetch a key at a time")
	flag.IntVar(&WATCHERS, "watchers", 0, "number of concurrent watchers")
	flag.StringVar(&MACHINES, "machines", "", "comma-separated list of etcd machines")
	flag.Parse()

	if (WRITERS == 0 && HEAVY_READERS == 0 && LIGHT_READERS == 0 && WATCHERS == 0) || MACHINES == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	etcdFlood := flood.NewFlood(STORE_SIZE, WRITERS, HEAVY_READERS, LIGHT_READERS, WATCHERS, strings.Split(MACHINES, ","))
	etcdFlood.Flood()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	<-c
	etcdFlood.Stop()
}
