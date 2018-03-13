package pit

import "fmt"

// PIT counters.
type Counters struct {
	NEntries  uint64 // current number of entries
	NInsert   uint64 // how many inserts created a new PIT entry
	NFound    uint64 // how many inserts found an existing PIT entry
	NCsMatch  uint64 // how many inserts matched a CS entry
	NAllocErr uint64 // how many inserts failed due to allocation error
	NExpired  uint64 // how many entries expired
}

func (cnt Counters) String() string {
	return fmt.Sprintf("%d entries, %d inserts, %d found, %d cs-match, %d alloc-err, %d expired",
		cnt.NEntries, cnt.NInsert, cnt.NFound, cnt.NCsMatch, cnt.NAllocErr, cnt.NExpired)
}

// Read PIT counters.
func (pit Pit) ReadCounters() (cnt Counters) {
	pitp := pit.getPriv()
	cnt.NEntries = uint64(pitp.nEntries)
	cnt.NInsert = uint64(pitp.nInsert)
	cnt.NFound = uint64(pitp.nFound)
	cnt.NCsMatch = uint64(pitp.nCsMatch)
	cnt.NAllocErr = uint64(pitp.nAllocErr)
	cnt.NExpired = uint64(pitp.timeoutSched.nTriggered)
	return cnt
}