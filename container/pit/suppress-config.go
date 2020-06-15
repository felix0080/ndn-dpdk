package pit

/*
#include "../../csrc/pcct/pit-suppress-config.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// PIT suppression configuration.
type SuppressConfig struct {
	Min        nnduration.Nanoseconds
	Max        nnduration.Nanoseconds
	Multiplier float64
}

func (sc SuppressConfig) CopyToC(ptr unsafe.Pointer) {
	c := (*C.PitSuppressConfig)(ptr)
	c.min = C.TscDuration(eal.ToTscDuration(sc.Min.DurationOr(10e6)))
	c.max = C.TscDuration(eal.ToTscDuration(sc.Max.DurationOr(100e6)))
	if sc.Multiplier < 1.0 {
		c.multiplier = 2.0
	} else {
		c.multiplier = C.double(sc.Multiplier)
	}
}
