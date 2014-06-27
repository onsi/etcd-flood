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
		node0 = StartNode(V44, Name(0), DataDir(0, true), Addr(0), PeerAddr(0), Peers(), "-snapshot-count=1000")
		node1 = StartNode(V44, Name(1), DataDir(1, true), Addr(1), PeerAddr(1), Peers(0, 2), "-snapshot-count=1000")
		node2 = StartNode(V44, Name(2), DataDir(2, true), Addr(2), PeerAddr(2), Peers(0, 1), "-snapshot-count=1000")

		flood = NewETCDFlood(1000, 50, Machines(0, 1, 2))
		flood.Flood()
		time.Sleep(10 * time.Second)
	})

	It("should work when the first node is shut down", func() {
		GreenBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(1)).Should(Equal(1000))
		Ω(KeysOnNode(2)).Should(Equal(1000))
	})

	It("should work when the first node comes back", func() {
		GreenBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		GreenBanner("restarting node 0...")
		node0 = StartNode(V44, Name(0), DataDir(0, false), Addr(0), PeerAddr(0), Peers(), "-snapshot-count=1000")
		time.Sleep(5 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(1000))
		Ω(KeysOnNode(1)).Should(Equal(1000))
		Ω(KeysOnNode(2)).Should(Equal(1000))
	})

	It("should work when the first node comes back to an empty directory", func() {
		GreenBanner("shutting down node 0")
		node0.Interrupt().Wait()
		time.Sleep(5 * time.Second)

		GreenBanner("removing data directory")
		err := os.RemoveAll(DataDir(0, false))
		Ω(err).ShouldNot(HaveOccurred())

		GreenBanner("restarting node 0...")
		node0 = StartNode(V44, Name(0), DataDir(0, true), Addr(0), PeerAddr(0), Peers(), "-snapshot-count=1000")
		time.Sleep(5 * time.Second)

		GreenBanner("checking...")
		Ω(KeysOnNode(0)).Should(Equal(1000))
		Ω(KeysOnNode(1)).Should(Equal(1000))
		Ω(KeysOnNode(2)).Should(Equal(1000))
	})
})
