package etcd_flood_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/etcd-flood"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

const V3 = "v0.3"
const V46 = "v0.4.6"
const VBETA = "vbeta"
const DATA_DIR = "./data-dir"

var toShutDown []*gexec.Session
var flood *ETCDFlood

var VERSION string
var STORE_SIZE int
var CONCURRENCY int
var READERS int
var WATCHERS int

func init() {
	flag.StringVar(&VERSION, "version", VBETA, "version to test: v0.3, v0.4.6, vbeta")
	flag.IntVar(&STORE_SIZE, "storeSize", 30000, "total number of keys to put in the store")
	flag.IntVar(&CONCURRENCY, "concurrency", 300, "number of concurrent requests")
	flag.IntVar(&READERS, "readers", 2, "number of concurrent readers")
	flag.IntVar(&WATCHERS, "watchers", 0, "number of concurrent watchers")
}

func TestEtcdFlood(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EtcdFlood Suite")
}

var _ = BeforeSuite(func() {
	err := os.MkdirAll(DATA_DIR, 0700)
	Ω(err).ShouldNot(HaveOccurred())
	for _, version := range []string{V3, V46, VBETA} {
		dir, err := filepath.Abs(filepath.Join("etcd", version))
		Ω(err).ShouldNot(HaveOccurred())

		cmd := exec.Command(filepath.Join(dir, "download.sh"))
		cmd.Dir = dir

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		Eventually(session, 60).Should(gexec.Exit(0))
	}
})

var _ = BeforeEach(func() {
	os.RemoveAll(DATA_DIR)
	err := os.MkdirAll(DATA_DIR, 0700)
	Ω(err).ShouldNot(HaveOccurred())

	toShutDown = []*gexec.Session{}
	flood = nil
})

var _ = AfterEach(func() {
	if flood != nil {
		flood.Stop()
	}

	for _, session := range toShutDown {
		session.Kill().Wait()
	}
})

var _ = AfterSuite(func() {
	for _, session := range toShutDown {
		session.Kill().Wait()
	}

	err := os.RemoveAll(DATA_DIR)
	Ω(err).ShouldNot(HaveOccurred())
})

func Addr(node int) string {
	return fmt.Sprintf("127.0.0.1:%d", 4001+node)
}

func PeerAddr(node int) string {
	return fmt.Sprintf("127.0.0.1:%d", 7001+node)
}

func Name(node int) string {
	return fmt.Sprintf("node-%d", node)
}

func DataDir(node int, create bool) string {
	dataDir := filepath.Join(DATA_DIR, Name(node))
	if create {
		os.MkdirAll(dataDir, 0700)
	}
	return dataDir
}

func Machines(nodes ...int) []string {
	machines := []string{}
	for _, node := range nodes {
		machines = append(machines, "http://"+Addr(node))
	}
	return machines
}

func Peers(nodes ...int) []string {
	peerAddrs := []string{}
	for _, node := range nodes {
		peerAddrs = append(peerAddrs, PeerAddr(node))
	}
	return peerAddrs
}

func StartNode(version string, name string, dataDir string, addr string, peerAddr string, peers []string, extraArgs ...string) *gexec.Session {
	args := []string{
		fmt.Sprintf("-name=%s", name),
		fmt.Sprintf("-addr=%s", addr),
		fmt.Sprintf("-peer-addr=%s", peerAddr),
		fmt.Sprintf("-data-dir=%s", dataDir),
		"-peer-heartbeat-timeout=50",
		"-peer-election-timeout=1000",
	}

	if len(peers) > 0 {
		args = append(args, fmt.Sprintf("-peers=%s", strings.Join(peers, ",")))
	}

	args = append(args, extraArgs...)

	path, err := filepath.Abs(filepath.Join("etcd", version, "etcd"))
	Ω(err).ShouldNot(HaveOccurred())

	cmd := exec.Command(path, args...)

	GreenBanner(fmt.Sprintf("Launching etcd %s [%s] with args:\n%s", version, name, format.IndentString(strings.Join(args, "\n"), 1)))

	session, err := gexec.Start(cmd,
		gexec.NewPrefixedWriter(fmt.Sprintf("[%s]", name), GinkgoWriter),
		gexec.NewPrefixedWriter(fmt.Sprintf("[%s]", name), GinkgoWriter))
	Ω(err).ShouldNot(HaveOccurred())
	toShutDown = append(toShutDown, session)

	WaitFor(addr)

	return session
}

func WaitFor(addr string) {
	client := &http.Client{
		Timeout: time.Second,
	}

	Eventually(func() int {
		resp, err := client.Get(fmt.Sprintf("http://%s/v2/stats/self", addr))
		if err != nil {
			return http.StatusInternalServerError
		}
		return resp.StatusCode
	}, 5).Should(Equal(http.StatusOK))
}

func KeysOnNode(node int) int {
	client := &http.Client{
		Timeout: time.Second,
	}

	response, err := client.Get(fmt.Sprintf("http://%s/v2/keys/flood?recursive=true", Addr(node)))
	Ω(err).ShouldNot(HaveOccurred())
	defer response.Body.Close()

	etcdResponse := etcd.Response{}
	err = json.NewDecoder(response.Body).Decode(&etcdResponse)
	Ω(err).ShouldNot(HaveOccurred())

	Ω(etcdResponse.Node).ShouldNot(BeNil())
	Ω(etcdResponse.Node.Dir).Should(BeTrue())
	return len(etcdResponse.Node.Nodes)
}
