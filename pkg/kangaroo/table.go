package kangaroo

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ajejfiejof/kangaroo-secp256k1/pkg/secp256k1"
)

// TrapEntry represents a recorded distinguished point in the trap table.
type TrapEntry struct {
	Y      secp256k1.FieldVal
	Scalar *big.Int
}

const numShards = 256

type trapShard struct {
	sync.RWMutex
	entries map[secp256k1.FieldVal]TrapEntry
}

// TrapTable is a concurrent sharded hash table for distinguished points.
type TrapTable struct {
	shards [numShards]trapShard
	count  uint64
}

// NewTrapTable allocates a sharded trap table.
func NewTrapTable() *TrapTable {
	tt := &TrapTable{}
	for i := 0; i < numShards; i++ {
		tt.shards[i].entries = make(map[secp256k1.FieldVal]TrapEntry)
	}
	return tt
}

func (tt *TrapTable) shardIndex(x secp256k1.FieldVal) int {
	return int(x[0] % numShards)
}

// Insert registers a distinguished point into the table.
func (tt *TrapTable) Insert(x, y secp256k1.FieldVal, scalar *big.Int) {
	idx := tt.shardIndex(x)

	tt.shards[idx].Lock()
	if _, exists := tt.shards[idx].entries[x]; !exists {
		tt.shards[idx].entries[x] = TrapEntry{
			Y:      y,
			Scalar: new(big.Int).Set(scalar),
		}
		atomic.AddUint64(&tt.count, 1)
	}
	tt.shards[idx].Unlock()
}

// Lookup tests whether a point's X coordinate exists in the table.
func (tt *TrapTable) Lookup(x secp256k1.FieldVal) (TrapEntry, bool) {
	idx := tt.shardIndex(x)

	tt.shards[idx].RLock()
	entry, exists := tt.shards[idx].entries[x]
	tt.shards[idx].RUnlock()

	return entry, exists
}

// Size returns total distinguished points stored.
func (tt *TrapTable) Size() uint64 {
	return atomic.LoadUint64(&tt.count)
}
