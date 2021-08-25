package set

import (
	"sync"
	"sync/atomic"
)

const (
	// platform bit = 2^setBits,(32/64)
	setBits  = 5 //+ (^uint(0) >> 63)
	platform = 1 << setBits
	setMesk  = 1<<setBits - 1
)

// IntSet is a set of non-negative integers.
// Its zero value represents the empty set.
//
// x is an item in set.
// x = (2^setBits)*idx + mod <==> x = 64*idx + mod  or  x = 32*idx + mod
// idx = x/2^setBits (x>>setBits) , mod = x%2^setBits (x&setMesk)
// in the set, x is the pesition: dirty[idx]&(1<<mod)
type IntSet struct {
	mu sync.Mutex

	// len(dirty)
	len uint32

	// only increase
	dirty []uint32
}

// in 64 bit platform
// x = 64*idx + mod
// idx = x/64 (x>>6) , mod = x%64 (x&(1<<6-1))
//
// in 32 bit platform
// x = 32*idx + mod
// idx = x/32 (x>>5) , mod = x%32 (x&(1<<5-1))
func idxMod(x uint32) (idx, mod uint32) {
	return x >> setBits, x & setMesk
}

// Load reports whether the set contains the non-negative value x.
func (s *IntSet) Load(x uint32) bool {
	idx, mod := idxMod(x)
	if idx >= atomic.LoadUint32(&s.len) {
		// overflow
		return false
	}
	item := atomic.LoadUint32(&s.dirty[idx])
	// return s.dirty[idx]&(1<<mod) != 0
	return (item>>mod)&1 == 1
}

// Store adds the non-negative value x to the set.
func (s *IntSet) Store(x uint32) {
	s.LoadOrStore(x)
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set
func (s *IntSet) LoadOrStore(x uint32) (loaded bool) {
	idx, mod := idxMod(x)
	if idx >= atomic.LoadUint32(&s.len) {
		s.mu.Lock()
		for {
			if idx < atomic.LoadUint32(&s.len) {
				break
			}
			s.dirty = append(s.dirty, 0)
			atomic.AddUint32(&s.len, 1)
		}
		s.mu.Unlock()
	}

	for {
		item := atomic.LoadUint32(&s.dirty[idx])
		if (item>>mod)&1 == 1 {
			return true
		}
		if atomic.CompareAndSwapUint32(&s.dirty[idx], item, item|(1<<mod)) {
			return false
		}
	}
}

// Delete remove x from the set
func (s *IntSet) Delete(x uint32) {
	s.LoadAndDelete(x)
}

// LoadAndDelete remove x from the set
// loaded report x if in set
func (s *IntSet) LoadAndDelete(x uint32) (loaded bool) {
	idx, mod := idxMod(x)
	if idx >= atomic.LoadUint32(&s.len) {
		// overflow
		return false
	}
	for {
		item := atomic.LoadUint32(&s.dirty[idx])
		if (item>>mod)&1 == 0 {
			return false
		}
		if atomic.CompareAndSwapUint32(&s.dirty[idx], item, item&^(1<<mod)) {
			return true
		}
	}
}

// Adds add all x in args to the set
func (s *IntSet) Adds(args ...uint32) {
	for _, x := range args {
		s.Store(x)
	}
}

// Removes remove all x in args to the set
func (s *IntSet) Removes(args ...uint32) {
	for _, x := range args {
		s.Delete(x)
	}
}
