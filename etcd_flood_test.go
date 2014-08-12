package etcd_flood_test

import (
	"os"
	"time"

	. "github.com/onsi/etcd-flood"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("When rolling etcd", func() {
	var node0, node1, node2, node3 *gexec.Session

	BeforeEach(func() {
		VERSION = V46
		node0 = StartNode(VERSION, Name(0), DataDir(0, true), Addr(0), PeerAddr(0), Peers(), "-snapshot-count=10000")
		node1 = StartNode(VERSION, Name(1), DataDir(1, true), Addr(1), PeerAddr(1), Peers(0, 2), "-snapshot-count=10000")
		node2 = StartNode(VERSION, Name(2), DataDir(2, true), Addr(2), PeerAddr(2), Peers(0, 1), "-snapshot-count=10000")

		flood = NewETCDFlood(NUM_KEYS, CONCURRENCY, Machines(0, 1, 2))
		flood.Flood()
		time.Sleep(10 * time.Second)
	})

	Context("the happy path", func() {
		It("should work when nothing messes with it", func() {
			GreenBanner("checking...")
			Ω(KeysOnNode(0)).Should(Equal(NUM_KEYS))
			Ω(KeysOnNode(1)).Should(Equal(NUM_KEYS))
			Ω(KeysOnNode(2)).Should(Equal(NUM_KEYS))
		})
	})

	It("should work when the first node is shut down", func() {
		GreenBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(1)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(2)).Should(Equal(NUM_KEYS))
	})

	It("should work when the first node comes back", func() {
		GreenBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		GreenBanner("restarting node 0...")
		node0 = StartNode(VERSION, Name(0), DataDir(0, false), Addr(0), PeerAddr(0), Peers(), "-snapshot-count=10000")
		time.Sleep(5 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(1)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(2)).Should(Equal(NUM_KEYS))
	})

	It("should work when a node joins the cluster", func() {
		GreenBanner("bringing up node 3")
		node3 = StartNode(VERSION, Name(3), DataDir(3, true), Addr(3), PeerAddr(3), Peers(0, 1, 2), "-snapshot-count=10000")

		GreenBanner("sleeping")
		time.Sleep(10 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(1)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(2)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(3)).Should(Equal(NUM_KEYS))
	})

	It("should work when the first node comes back to an empty directory", func() {
		GreenBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		GreenBanner("removing data directory")
		err := os.RemoveAll(DataDir(0, false))
		Ω(err).ShouldNot(HaveOccurred())

		GreenBanner("restarting node 0...")
		node0 = StartNode(VERSION, Name(0), DataDir(0, true), Addr(0), PeerAddr(0), Peers(), "-snapshot-count=10000")
		time.Sleep(5 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(1)).Should(Equal(NUM_KEYS))
		Ω(KeysOnNode(2)).Should(Equal(NUM_KEYS))
	})
})
