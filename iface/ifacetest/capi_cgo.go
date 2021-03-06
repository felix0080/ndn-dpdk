package ifacetest

/*
#include "../../csrc/iface/face.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func Face_IsDown(faceId iface.FaceId) bool {
	return bool(C.Face_IsDown(C.FaceId(faceId)))
}

func Face_TxBurst(faceId iface.FaceId, pkts []*ndni.Packet) {
	ptr, count := cptr.ParseCptrArray(pkts)
	C.Face_TxBurst(C.FaceId(faceId), (**C.Packet)(ptr), C.uint16_t(count))
}
