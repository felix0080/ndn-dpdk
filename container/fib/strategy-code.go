package fib

/*
#include "strategy-code.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// BPF program of a forwarding strategy.
type StrategyCode struct {
	c *C.StrategyCode
}

func (sc StrategyCode) GetId() int {
	return int(sc.c.id)
}

func (sc StrategyCode) CountRefs() int {
	return int(sc.c.nRefs)
}

func (sc StrategyCode) Close() error {
	if sc.CountRefs() > 0 {
		return errors.New("StrategyCode has references")
	}

	strategyCodeMutex.Lock()
	defer strategyCodeMutex.Unlock()
	C.ubpf_destroy(sc.c.vm)
	delete(strategyCodeTable, sc.GetId())
	dpdk.Free(sc.c)
	return nil
}

func (sc StrategyCode) String() string {
	if sc.c == nil {
		return "0@nil"
	}
	return fmt.Sprintf("%d@%p", sc.GetId(), sc.c)
}

// Table of StrategyCode instances.
var (
	strategyCodeId    int
	strategyCodeTable map[int]StrategyCode = make(map[int]StrategyCode)
	strategyCodeMutex sync.Mutex
)

// A global function that registers CALL-able functions available to BPF program.
// fib.LoadStrategyCode is available only if this is specified.
var RegisterStrategyFuncs func(vm unsafe.Pointer) error

func fromUbpfError(funcName string, errC *C.char) error {
	err := C.GoString(errC)
	C.free(unsafe.Pointer(errC))
	return fmt.Errorf("%s: %v", funcName, err)
}

func makeStrategyCode(load func(vm *C.struct_ubpf_vm) error) (sc StrategyCode, e error) {
	vm := C.ubpf_create()
	if vm == nil {
		return sc, errors.New("ubpf_create failed")
	}

	if e = load(vm); e != nil {
		C.ubpf_destroy(vm)
		return sc, e
	}

	var errC *C.char
	jit := C.ubpf_compile(vm, &errC)
	if jit == nil {
		C.ubpf_destroy(vm)
		return sc, fromUbpfError("ubpf_compile", errC)
	}

	strategyCodeMutex.Lock()
	defer strategyCodeMutex.Unlock()
	strategyCodeId++
	sc.c = (*C.StrategyCode)(dpdk.Zmalloc("StrategyCode", C.sizeof_StrategyCode, dpdk.NUMA_SOCKET_ANY))
	sc.c.id = C.int(strategyCodeId)
	sc.c.vm = vm
	sc.c.jit = jit
	strategyCodeTable[strategyCodeId] = sc
	return sc, nil
}

// Load a strategy BPF program from ELF object.
func LoadStrategyCode(elf []byte) (sc StrategyCode, e error) {
	if RegisterStrategyFuncs == nil {
		return sc, errors.New("fib.RegisterStrategyFuncs is empty")
	}

	return makeStrategyCode(func(vm *C.struct_ubpf_vm) error {
		e = RegisterStrategyFuncs(unsafe.Pointer(vm))
		if e != nil {
			return e
		}
		var errC *C.char
		if res := C.ubpf_load_elf(vm, unsafe.Pointer(&elf[0]), C.size_t(len(elf)), &errC); res != 0 {
			return fromUbpfError("ubpf_load_elf", errC)
		}
		return nil
	})
}

// Load an empty BPF program (mainly for unit testing).
func MakeEmptyStrategy() (sc StrategyCode, e error) {
	return makeStrategyCode(func(vm *C.struct_ubpf_vm) error {
		code := []uint64{0x95}
		var errC *C.char
		if res := C.ubpf_load(vm, unsafe.Pointer(&code[0]), 8, &errC); res != 0 {
			return fromUbpfError("ubpf_load", errC)
		}
		return nil
	})
}

func GetStrategyCode(id int) (sc StrategyCode, e error) {
	strategyCodeMutex.Lock()
	defer strategyCodeMutex.Unlock()
	var ok bool
	if sc, ok = strategyCodeTable[id]; !ok {
		return sc, fmt.Errorf("StrategyCode(%d) does not exist", id)
	}
	return sc, nil
}

func ListStrategyCode() []StrategyCode {
	strategyCodeMutex.Lock()
	defer strategyCodeMutex.Unlock()
	list := make([]StrategyCode, 0, len(strategyCodeTable))
	for _, sc := range strategyCodeTable {
		list = append(list, sc)
	}
	return list
}

func UnloadAllStrategyCode() {
	for _, sc := range ListStrategyCode() {
		sc.Close()
	}
}
