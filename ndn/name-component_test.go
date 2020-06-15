package ndn_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestNameComponent(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input string
		bad   bool
		t     uint64
		v     string
		str   string
	}{
		{input: "FD01", bad: true},                                             // incomplete TLV-TYPE
		{input: "08FD01", bad: true},                                           // incomplete TLV-LENGTH
		{input: "0802B0", bad: true},                                           // incomplete TLV-VALUE
		{input: "0802B0B1B2", bad: true},                                       // junk at end
		{input: "0001B0", bad: true},                                           // TLV-TYPE too small
		{input: "FE00010000", bad: true},                                       // TLV-TYPE too large
		{input: "0104B0B1B2B3", t: 0x01, v: "B0B1B2B3", str: "1=%B0%B1%B2%B3"}, // ImplicitDigest wrong TLV-LENGTH

		{input: "0800", t: 0x08, v: "", str: "8=..."},
		{input: "08012E", t: 0x08, v: "2E", str: "8=...."},
		{input: "0200", t: 0x02, v: "", str: "2=..."},
		{input: "FC00", t: 0xFC, v: "", str: "252=..."},
		{input: "FDFFFF00", t: 0xFFFF, v: "", str: "65535=..."},
		{input: "96012E", t: 0x96, v: "2E", str: "150=...."},
		{input: "08052D2E5F7E41", t: 0x08, v: "2D2E5F7E41", str: "8=-._~A"},
		{input: "0804002081FF", t: 0x08, v: "002081FF", str: "8=%00%20%81%FF"},
		{input: "0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)", t: 0x01,
			v:   "DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A",
			str: "1=%DCmh%40%C6%FA%FBw%3DX%3C%DB%F4ef%1C%7BK%96%8E%04%AC%D4%D9%01%5B%1CNS%E5%9Dj"},
	}

	for _, tt := range tests {
		comp := ndn.NameComponent(mbuftestenv.BytesFromHex(tt.input))
		if tt.bad {
			assert.False(comp.IsValid(), tt.input)
		} else if assert.True(comp.IsValid(), tt.input) {
			assert.Equal(ndn.TlvType(tt.t), comp.GetType(), tt.input)
			assert.Equal(ndn.TlvBytes(mbuftestenv.BytesFromHex(tt.v)), comp.GetValue(), tt.input)
			assert.Equal(tt.str, comp.String(), tt.input)

			parsed, e := ndn.ParseNameComponent(tt.str)
			if assert.NoError(e, tt.input) {
				assert.True(comp.Equal(parsed))
			}
		}
	}
}
