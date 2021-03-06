package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
#include "../../csrc/fwdp/strategy.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Forwarding thread.
type Fwd struct {
	eal.ThreadBase
	id            int
	c             *C.FwFwd
	pcct          *pcct.Pcct
	interestQueue *pktqueue.PktQueue
	dataQueue     *pktqueue.PktQueue
	nackQueue     *pktqueue.PktQueue
}

func newFwd(id int) *Fwd {
	var fwd Fwd
	fwd.id = id
	return &fwd
}

func (fwd *Fwd) String() string {
	return fmt.Sprintf("fwd%d", fwd.id)
}

func (fwd *Fwd) Init(fib *fib.Fib, pcctCfg pcct.Config, interestQueueCfg, dataQueueCfg, nackQueueCfg pktqueue.Config,
	latencySampleFreq int, suppressCfg pit.SuppressConfig) (e error) {
	socket := fwd.GetNumaSocket()

	fwd.c = (*C.FwFwd)(eal.Zmalloc("FwFwd", C.sizeof_FwFwd, socket))
	eal.InitStopFlag(unsafe.Pointer(&fwd.c.stop))
	fwd.c.id = C.uint8_t(fwd.id)

	if fwd.interestQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inInterestQueue), interestQueueCfg, fmt.Sprintf("%s_qI", fwd), socket); e != nil {
		return nil
	}
	if fwd.dataQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inDataQueue), dataQueueCfg, fmt.Sprintf("%s_qD", fwd), socket); e != nil {
		return nil
	}
	if fwd.nackQueue, e = pktqueue.NewAt(unsafe.Pointer(&fwd.c.inNackQueue), nackQueueCfg, fmt.Sprintf("%s_qN", fwd), socket); e != nil {
		return nil
	}

	fwd.c.fib = (*C.Fib)(fib.GetPtr(fwd.id))

	pcctCfg.Socket = socket
	fwd.pcct, e = pcct.New(fwd.String()+"_pcct", pcctCfg)
	if e != nil {
		return fmt.Errorf("pcct.New: %v", e)
	}
	*C.FwFwd_GetPcctPtr_(fwd.c) = (*C.Pcct)(fwd.pcct.GetPtr())

	fwd.c.headerMp = (*C.struct_rte_mempool)(ndni.HeaderMempool.MakePool(socket).GetPtr())
	fwd.c.guiderMp = (*C.struct_rte_mempool)(ndni.NameMempool.MakePool(socket).GetPtr())
	fwd.c.indirectMp = (*C.struct_rte_mempool)(pktmbuf.Indirect.MakePool(socket).GetPtr())

	latencyStat := runningstat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	latencyStat.Clear(false)
	latencyStat.SetSampleRate(latencySampleFreq)

	suppressCfg.CopyToC(unsafe.Pointer(&fwd.c.suppressCfg))

	return nil
}

func (fwd *Fwd) Launch() error {
	return fwd.LaunchImpl(func() int {
		rs := urcu.NewReadSide()
		defer rs.Close()
		C.FwFwd_Run(fwd.c)
		return 0
	})
}

func (fwd *Fwd) Stop() error {
	return fwd.StopImpl(eal.NewStopFlag(unsafe.Pointer(&fwd.c.stop)))
}

func (fwd *Fwd) Close() error {
	fwd.interestQueue.Close()
	fwd.dataQueue.Close()
	fwd.nackQueue.Close()
	fwd.pcct.Close()
	eal.Free(fwd.c)
	return nil
}

func init() {
	var nXsyms C.int
	strategycode.Xsyms = unsafe.Pointer(C.SgGetXsyms(&nXsyms))
	strategycode.NXsyms = int(nXsyms)
}
