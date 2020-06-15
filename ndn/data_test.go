package ndn_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestDataDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		name      string
		freshness int
	}{
		{input: "0600", bad: true},                        // missing Name
		{input: "0604 meta=1400 content=1500", bad: true}, // missing Name
		{input: "0602 name=0700", name: "/"},
		{input: "0605 name=0703080141", name: "/A"},
		{input: "0615 name=0703080142 meta=140C (180102 fp=190201FF 1A03080142) content=1500", name: "/B", freshness: 0x01FF},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsMbuf().Close()
		e := pkt.ParseL3(ndntestenv.Name.Pool())
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndn.L3PktType_Data, pkt.GetL3Type(), tt.input) {
				continue
			}
			data := pkt.AsData()
			assert.Implements((*ndn.IL3Packet)(nil), data)
			ndntestenv.NameEqual(assert, tt.name, data, tt.input)
			assert.EqualValues(tt.freshness, data.GetFreshnessPeriod()/time.Millisecond, tt.input)
		}
	}
}

func TestDataSatisfy(t *testing.T) {
	assert, _ := makeAR(t)

	interestExact := makeInterest("/B")
	interestPrefix := makeInterest("/B", ndn.CanBePrefixFlag)
	interestFresh := makeInterest("/B", ndn.MustBeFreshFlag)

	tests := []struct {
		data        *ndn.Data
		exactMatch  ndn.DataSatisfyResult
		prefixMatch ndn.DataSatisfyResult
		freshMatch  ndn.DataSatisfyResult
	}{
		{makeData("/A", time.Second),
			ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO},
		{makeData("/2=B", time.Second),
			ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO},
		{makeData("/B", time.Second),
			ndn.DATA_SATISFY_YES, ndn.DATA_SATISFY_YES, ndn.DATA_SATISFY_YES},
		{makeData("/B", time.Duration(0)),
			ndn.DATA_SATISFY_YES, ndn.DATA_SATISFY_YES, ndn.DATA_SATISFY_NO},
		{makeData("/B/0", time.Second),
			ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_YES, ndn.DATA_SATISFY_NO},
		{makeData("/", time.Second),
			ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO},
		{makeData("/C", time.Second),
			ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO, ndn.DATA_SATISFY_NO},
	}
	for i, tt := range tests {
		assert.Equal(tt.exactMatch, tt.data.CanSatisfy(interestExact), "%d", i)
		assert.Equal(tt.prefixMatch, tt.data.CanSatisfy(interestPrefix), "%d", i)
		assert.Equal(tt.freshMatch, tt.data.CanSatisfy(interestFresh), "%d", i)

		if tt.exactMatch == ndn.DATA_SATISFY_YES {
			interestImplicit := makeInterest(tt.data.GetFullName().String())
			assert.Equal(ndn.DATA_SATISFY_NEED_DIGEST, tt.data.CanSatisfy(interestImplicit))
			tt.data.ComputeDigest(true)
			assert.Equal(ndn.DATA_SATISFY_YES, tt.data.CanSatisfy(interestImplicit))
			ndntestenv.ClosePacket(interestImplicit)
		}

		ndntestenv.ClosePacket(tt.data)
	}

	ndntestenv.ClosePacket(interestExact)
	ndntestenv.ClosePacket(interestPrefix)
	ndntestenv.ClosePacket(interestFresh)
}

func TestDataDigest(t *testing.T) {
	assert, require := makeAR(t)

	cd, e := cryptodev.MultiSegDrv.Create("", cryptodev.Config{}, eal.NumaSocket{})
	require.NoError(e)
	defer cd.Close()
	qp := cd.GetQueuePair(0)
	mp, e := cryptodev.NewOpPool("MP-CryptoOp", cryptodev.OpPoolConfig{}, eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	names := []string{
		"/",
		"/A",
		"/B",
		"/C",
	}
	inputs := make([]*ndn.Data, 4)
	for i, name := range names {
		inputs[i] = makeData(name)
	}

	ops, e := mp.Alloc(cryptodev.OpSymmetric, 4)
	require.NoError(e)
	for i, data := range inputs {
		assert.Nil(data.GetDigest())
		data.DigestPrepare(ops[i])
	}
	assert.Equal(4, qp.EnqueueBurst(ops))

	assert.Equal(4, qp.DequeueBurst(ops))
	for i, op := range ops {
		data, e := ndn.DataDigest_Finish(op)
		assert.NoError(e)
		if assert.NotNil(data) {
			ndntestenv.NameEqual(assert, names[i], data)
			assert.Equal(data.ComputeDigest(false), data.GetDigest())
		}
	}
}
