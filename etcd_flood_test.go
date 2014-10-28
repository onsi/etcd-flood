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
	var node0, node1, node2 *gexec.Session

	BeforeEach(func() {
		node0 = StartNode(VERSION, 3, 0, DataDir(0, true), "-snapshot-count=1000")
		node1 = StartNode(VERSION, 3, 1, DataDir(1, true), "-snapshot-count=1000")
		node2 = StartNode(VERSION, 3, 2, DataDir(2, true), "-snapshot-count=1000")

		flood = NewETCDFlood(STORE_SIZE, CONCURRENCY, HEAVY_READERS, LIGHT_READERS, WATCHERS, Machines(0, 1, 2))
		flood.Flood()
		time.Sleep(10 * time.Second)
	})

	Context("the happy path", func() {
		It("should work when nothing messes with it", func() {
			YellowBanner("checking...")
			Ω(KeysOnNode(0)).Should(Equal(STORE_SIZE))
			Ω(KeysOnNode(1)).Should(Equal(STORE_SIZE))
			Ω(KeysOnNode(2)).Should(Equal(STORE_SIZE))
		})
	})

	It("should work when the first node is shut down", func() {
		YellowBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		YellowBanner("checking...")
		Ω(KeysOnNode(1)).Should(Equal(STORE_SIZE))
		Ω(KeysOnNode(2)).Should(Equal(STORE_SIZE))
	})

	It("should work when the first node comes back", func() {
		YellowBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		YellowBanner("restarting node 0...")
		node0 = StartNode(VERSION, 3, 0, DataDir(0, false), "-snapshot-count=1000")
		time.Sleep(5 * time.Second)

		YellowBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(STORE_SIZE))
		Ω(KeysOnNode(1)).Should(Equal(STORE_SIZE))
		Ω(KeysOnNode(2)).Should(Equal(STORE_SIZE))
	})

	It("should work when the first node comes back to an empty directory", func() {
		YellowBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		YellowBanner("removing data directory")
		err := os.RemoveAll(DataDir(0, false))
		Ω(err).ShouldNot(HaveOccurred())

		YellowBanner("restarting node 0...")
		node0 = StartNode(VERSION, 3, 0, DataDir(0, true), "-snapshot-count=1000")
		time.Sleep(5 * time.Second)

		YellowBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(STORE_SIZE))
		Ω(KeysOnNode(1)).Should(Equal(STORE_SIZE))
		Ω(KeysOnNode(2)).Should(Equal(STORE_SIZE))
	})
})
