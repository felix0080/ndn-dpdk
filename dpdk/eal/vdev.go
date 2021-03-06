package eal

/*
#include "../../csrc/core/common.h"
#include <rte_bus_vdev.h>
*/
import "C"
import (
	"unsafe"
)

// CreateVdev creates a virtual device.
func CreateVdev(name, args string) error {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	argsC := C.CString(args)
	defer C.free(unsafe.Pointer(argsC))

	if res := C.rte_vdev_init(nameC, argsC); res != 0 {
		return Errno(-res)
	}
	return nil
}

// DestroyVdev destroys a virtual device.
func DestroyVdev(name string) error {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))

	if res := C.rte_vdev_uninit(nameC); res != 0 {
		return Errno(-res)
	}
	return nil
}
