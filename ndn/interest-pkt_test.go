package ndn

import (
	"encoding/binary"
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestDecodeInterest(t *testing.T) {
	assert, require := makeAR(t)

	checkNonce := func(nonce uint32) func() bool {
		return func() bool {
			return nonce == 0xCACBCCCD || nonce == 0xCDCCCBCA
		}
	}

	tests := []struct {
		input       string
		ok          bool
		name        string
		mustBeFresh bool
		lifetime    int
		fwHints     []string
	}{
		{"", false, "", false, 0, nil},
		{"0508 name=0700 nonce=0A04CACBCCCD", true, "/", false, 4000, nil},
		{"050E name=0706080141080142 nonce=0A04CACBCCCD", true, "/A/B", false, 4000, nil},
		{"0515 name=0700 selectors=090B 0D0101 0E0101 110101 1200 nonce=0A04CACBCCCD", true,
			"/", true, 4000, nil},
		{"050B name=0700 nonce=0A04CACBCCCD lifetime=0C01FF", true, "/", false, 255, nil},
		{"0514 name=0700 nonce=0A04CACBCCCD fwhint=1E0A (del=1F08 pref=1E0100 name=0703080147)", true,
			"/", false, 4000, []string{"/G"}},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.Truef(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		interest, e := d.ReadInterest()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(tt.name, interest.GetName().String(), tt.input)
				assert.Equal(tt.mustBeFresh, interest.HasMustBeFresh(), tt.input)
				assert.Condition(checkNonce(interest.GetNonce()), tt.input)
				assert.EqualValues(tt.lifetime, interest.GetLifetime()/time.Millisecond, tt.input)

				fwHints := interest.GetFwHints()
				if assert.Len(fwHints, len(tt.fwHints), tt.input) {
					for i, fhName := range fwHints {
						assert.Equal(tt.fwHints[i], fhName.String(), tt.input, i)
					}
				}
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}

func checkEncodeInterest(t *testing.T, tpl *InterestTemplate, expectedHex string) {
	assert, _ := makeAR(t)

	expected := dpdktestenv.PacketBytesFromHex(expectedHex)

	pkt := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	tpl.EncodeTo(pkt)
	assert.Equal(len(expected), pkt.Len())

	actual := make([]byte, len(expected))
	pkt.ReadTo(0, actual)
	assert.NotEqual(binary.LittleEndian.Uint32(actual[len(expected)-4:]), uint32(0xCCCCCCCC))
	binary.LittleEndian.PutUint32(actual[len(expected)-4:], 0xCCCCCCCC)
	assert.Equal(expected, actual)
}

func TestEncodeInterest0(t *testing.T) {
	assert, _ := makeAR(t)

	tpl := NewInterestTemplate()
	e := tpl.SetNamePrefixFromUri("/")
	assert.NoError(e)
	tpl.SetMustBeFresh(false)
	assert.False(tpl.GetMustBeFresh())
	assert.Equal(4000*time.Millisecond, tpl.GetInterestLifetime())

	checkEncodeInterest(t, tpl, "050E 0700 0C0400000FA0 0A04CCCCCCCC")
}

func TestEncodeInterest1(t *testing.T) {
	assert, _ := makeAR(t)

	tpl := NewInterestTemplate()
	e := tpl.SetNamePrefixFromUri("/A/B")
	assert.NoError(e)
	tpl.SetMustBeFresh(true)
	assert.True(tpl.GetMustBeFresh())
	tpl.SetInterestLifetime(9000 * time.Millisecond)
	assert.Equal(9000*time.Millisecond, tpl.GetInterestLifetime())

	checkEncodeInterest(t, tpl, "0518 0706080141080142 09021200 0C0400002328 0A04CCCCCCCC")
}
