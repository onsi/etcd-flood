# ETCD Flood

A Ginkgo test suite that spins up an etcd cluster, writes to it and reads from it mercilessly, and then goes through a variety of excercises tearing down etcd nodes.

Currently, there are 5 tests:

- bring up 3 nodes, wait for a while, assert that all nodes have the correct data
- bring up nodes A B and C, shut A down, assert that B and C are fine
- bring up nodes A B and C, shut A down, then bring it back.  Assert that A, B and C are fine
- bring up nodes A B and C, shut A down, then delete its data directory, then bring it back.  Assert that A, B and C are fine

The last test sometimes fails with v0.3 and v0.4.6.  When it does, etcd is often logging:

```
WARNING   | [ss] Error: nil response
WARNING   | transporter.ss.decoding.error:proto: field/encoding mismatch: wrong type for field
```

This seems to be an issue with snapshotting.  To increase the likelihood of it happening we set the `-snapshot-count=1000`.

## Running the suite

```
go get github.com/onsi/gomega
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/etcd-flood
cd $GOPATH/src/github.com/onsi/etcd-flood
ginkgo -v
```

The test suite will automatically download etcd binaries from Github (assumes OS X).  This will only happen the first time.

To rerun the tests until failure occurs:

```
ginkgo -v -untilItFails
```

## Testing different versions of etcd

```
ginkgo -v -- -version=VERSION
```

where `VERSION` is one of `v0.3`, `v0.4.6`, or `v0.5`

`VERSION` defaults to `vbeta`.

## What the flood does

The flood spins up `WRITERS` concurrent writers that write to the etcd cluster in a tight loop.  The flood writes and maintains `STORE_SIZE` keys in etcd.  To be clear: the flood does *not* fill etcd up with keys, rather it fills it up to `STORE_SIZE` keys and then constantly overwrites those keys.

The flood will also *read* from the store.  There are two types of concurrent reads peformed.  The flood will spawn `LIGHT_READERS` readers that each read a single key from the store.  It will also spawn `HEAVY_READERS` readers that fetch the entire contents of the store.  Readers pause for 10ms between reads.


The default values are `WRITERS = 300`, `HEAVY_READERS = 5`, `LIGHT_READERS = 20` and `STORE_SIZE = 30000`.  These can be modified via:

```
ginkgo -v -- -writers=WRITERS -heavyReaders=HEAVY_REAERS -lightReaders=LIGHT_READERS -storeSize=STORE_SIZE
```

The flood then reports stats about the run: how many writes/reads succeeded, how many failed, how long they took, etc.  The test is allowed to succeed even if some writes fails.  The test only fails if, at the end of the run, the individual etcd nodes do not each report `30000` elements.

## Results

With these default values I generally see the happy-path test maintain `~2700 writes/second, ~35 heavy reads/second, 800 light reads/second` on a 3.4 GHz i7 iMac.

The sad path tests generally lead to a brief window of downtime in which etcd is unavailable.

## `etcd-flood`

`go get github.com/onsi/etcd-flood` also builds an `etcd-flood` binary that can be used to launch an etcd flood.  It has similar CLI flags as the test suite.
