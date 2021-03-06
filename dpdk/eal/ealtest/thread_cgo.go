package ealtest

/*
#include "../../../csrc/dpdk/thread.h"

typedef struct TestThread {
	int n;
	ThreadStopFlag stop;
} TestThread;

void
TestThread_Run(TestThread* thread) {
	thread->n = 0;
	while (ThreadStopFlag_ShouldContinue(&thread->stop)) {
		++thread->n;
	}
}
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

type testThread struct {
	eal.ThreadBase
	c *C.TestThread
}

func newTestThread() (th *testThread) {
	th = new(testThread)
	th.c = (*C.TestThread)(eal.Zmalloc("TestThread", C.sizeof_TestThread, eal.NumaSocket{}))
	eal.InitStopFlag(unsafe.Pointer(&th.c.stop))
	return th
}

func (th *testThread) GetN() int {
	return int(th.c.n)
}

func (th *testThread) Launch() error {
	return th.LaunchImpl(func() int {
		C.TestThread_Run(th.c)
		return 0
	})
}

func (th *testThread) Stop() error {
	return th.StopImpl(eal.NewStopFlag(unsafe.Pointer(&th.c.stop)))
}

func (th *testThread) Close() error {
	eal.Free(th.c)
	return nil
}
