package sockettransport_test

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

var trCfg = sockettransport.Config{
	RxQueueSize: 64,
	TxQueueSize: 64,
}

func TestPipe(t *testing.T) {
	_, require := makeAR(t)

	pipeA, pipeB := net.Pipe()
	trA, e := sockettransport.New(pipeA, trCfg)
	require.NoError(e)
	trB, e := sockettransport.New(pipeB, trCfg)
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}

func TestUdp(t *testing.T) {
	_, require := makeAR(t)

	var dialer sockettransport.Dialer
	dialer.Config = trCfg

	trA, e := dialer.Dial("udp", "127.0.0.1:7001", "127.0.0.1:7002")
	require.NoError(e)
	trB, e := dialer.Dial("udp", "127.0.0.1:7002", "127.0.0.1:7001")
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}

func TestTcp(t *testing.T) {
	_, require := makeAR(t)

	listener, e := net.Listen("tcp", "127.0.0.1:7002")
	require.NoError(e)
	defer listener.Close()

	checkStream(t, listener)
}

func TestUnix(t *testing.T) {
	_, require := makeAR(t)

	tmpdir, e := ioutil.TempDir("", "sockettransport-test")
	require.NoError(e)
	defer os.RemoveAll(tmpdir)
	addr := path.Join(tmpdir, "unix.sock")
	listener, e := net.Listen("unix", addr)
	require.NoError(e)
	defer listener.Close()

	checkStream(t, listener)
}

func checkStream(t *testing.T, listener net.Listener) {
	_, require := makeAR(t)

	var trA, trB *sockettransport.Transport
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		var dialer sockettransport.Dialer
		dialer.Config = trCfg
		listenAddr := listener.Addr()
		tr, e := dialer.Dial(listenAddr.Network(), "", listenAddr.String())
		require.NoError(e)
		trA = tr
		wg.Done()
	}()

	go func() {
		socket, e := listener.Accept()
		require.NoError(e)
		tr, e := sockettransport.New(socket, trCfg)
		require.NoError(e)
		trB = tr
		wg.Done()
	}()

	wg.Wait()

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}
