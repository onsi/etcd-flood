# ETCD Flood

A Ginkgo test suite that spins up an etcd cluster, writes to it mercilessly, and then goes through a variety of excercises tearing down etcd nodes.

Currently, there are three tests:

- bring up 3 nodes (A, B, C), shut A down, assert that B and C are fine.
- bring up 3 nodes (A, B, C), shut A down, then bring it back.  Assert that A and B and C are fine.
- bring up 3 nodes (A, B, C), shut A down then delete its data directory, then bring it back.  Assert that A and B and C are fine.

The last test currently fails (usually... though not always).  One sees A come back and spit out:

```
[ss] Error: nil response
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