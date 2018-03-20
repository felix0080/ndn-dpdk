# ndn-dpdk/iface/mockface

This package implements a mock face for testing.

Test code can invoke `MockFace.Rx` to cause the face to receive a packet.
All MockFaces depend on `MockFace.TheRxLoop` singleton as their `iface.IRxLooper`.

Packets transmitted through a mock face are accumulated on `MockFace.TxInterests`, `MockFace.TxData`, or `MockFace.TxNacks` slices.
Test code is responsible for freeing these packets.