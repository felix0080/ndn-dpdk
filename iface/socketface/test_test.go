package socketface_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
)

var socketfaceCfg socketface.Config

func TestMain(m *testing.M) {
	ealtestenv.InitEal()

	socketfaceCfg = socketface.Config{
		TxqPkts:   64,
		TxqFrames: 64,
	}

	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
