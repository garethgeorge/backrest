package oplog

import (
	rand "math/rand/v2"
	"sync"

	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

// setup a fast random number generator seeded with cryptographic randomness.
var mu sync.Mutex
var pgcRand = rand.NewPCG(cryptoutil.MustRandomUint64(), cryptoutil.MustRandomUint64())
var randGen = rand.New(pgcRand)

func NewRandomModno(lastModno int64) int64 {
	mu.Lock()
	defer mu.Unlock()
	for {
		modno := randGen.Int64()
		if modno != lastModno {
			return modno
		}
	}
}
