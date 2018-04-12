package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ndn-dpdk/app/dump"
	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

const (
	Dump_RingCapacity    = 256
	Face_TxQueueCapacity = 256
)

type PktcopyProc struct {
	face      iface.IFace
	pcrx      *PktcopyRx
	rxLcore   dpdk.LCore
	txLcore   dpdk.LCore
	dumper    *dump.Dump
	dumpLcore dpdk.LCore
}

func main() {
	appinit.InitEal()
	pc, e := ParseCommand(appinit.Eal.Args[1:])
	if e != nil {
		appinit.Exitf(appinit.EXIT_BAD_CONFIG, "parseCommand: %v", e)
	}

	// initialize faces, PktcopyRxs, and dumpers
	lcr := appinit.NewLCoreReservations()
	procs := make([]PktcopyProc, len(pc.Faces))
	for i, faceUri := range pc.Faces {
		proc := &procs[i]
		face, e := appinit.NewFaceFromUri(faceUri)
		if e != nil {
			appinit.Exitf(appinit.EXIT_FACE_INIT_ERROR, "NewFaceFromUri(%s): %v", faceUri, e)
		}
		face.EnableThreadSafeTx(Face_TxQueueCapacity)
		numaSocket := face.GetNumaSocket()
		proc.face = face

		pcrx := NewPktcopyRx(face)
		proc.pcrx = pcrx

		if pc.Dump {
			ringName := fmt.Sprintf("dump_%d", i)
			ring, e := dpdk.NewRing(ringName, Dump_RingCapacity, numaSocket, true, true)
			if e != nil {
				appinit.Exitf(appinit.EXIT_RING_INIT_ERROR, "NewRing(%s): %v", ringName, e)
			}
			pcrx.SetDumpRing(ring)

			prefix := fmt.Sprintf("%d ", face.GetFaceId())
			logger := log.New(os.Stderr, prefix, log.Lmicroseconds)
			proc.dumper = dump.New(ring, logger)
		}

		proc.rxLcore = lcr.ReserveRequired(numaSocket)
		proc.txLcore = lcr.ReserveRequired(numaSocket)
	}
	if pc.Dump {
		for i := range procs {
			procs[i].dumpLcore = lcr.ReserveRequired(dpdk.NUMA_SOCKET_ANY)
		}
	}

	// link PktcopyRx to TX faces
	switch pc.Mode {
	case TopoMode_Pair:
		for i := 0; i < len(procs); i += 2 {
			procs[i].pcrx.AddTxFace(procs[i+1].face)
			procs[i+1].pcrx.AddTxFace(procs[i].face)
		}
	case TopoMode_All:
		for i := range procs {
			for j := range procs {
				if i == j {
					continue
				}
				procs[i].pcrx.AddTxFace(procs[j].face)
			}
		}
	case TopoMode_OneWay:
		for i := 1; i < len(procs); i++ {
			procs[0].pcrx.AddTxFace(procs[i].face)
		}
	}

	// print counters
	tick := time.Tick(pc.CntInterval)
	go func() {
		for {
			<-tick
			for _, faceId := range iface.ListFaceIds() {
				log.Printf("%d %v", faceId, iface.Get(faceId).ReadCounters())
			}
		}
	}()

	// launch
	for _, proc := range procs {
		proc.txLcore.RemoteLaunch(func() int {
			appinit.MakeTxLooper(proc.face).TxLoop()
			return 0
		})
		if proc.dumper != nil {
			proc.dumpLcore.RemoteLaunch(proc.dumper.Run)
		}
	}
	for _, proc := range procs {
		proc.rxLcore.RemoteLaunch(proc.pcrx.Run)
	}

	select {}
}
