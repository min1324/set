package set

import (
	"bytes"
	"fmt"
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
// time complexity: O(1)
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
// time complexity: O(1)
func (s *IntSet) Store(x uint32) {
	s.LoadOrStore(x)
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set
// time complexity: O(1)
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
// time complexity: O(1)
func (s *IntSet) Delete(x uint32) {
	s.LoadAndDelete(x)
}

// LoadAndDelete remove x from the set
// loaded report x if in set
// time complexity: O(1)
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
// time complexity: O(n)
func (s *IntSet) Adds(args ...uint32) {
	for _, x := range args {
		s.Store(x)
	}
}

// Removes remove all x in args to the set
// time complexity: O(n)
func (s *IntSet) Removes(args ...uint32) {
	for _, x := range args {
		s.Delete(x)
	}
}

// Range calls f sequentially for each item present in the set.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the set's
// contents: no item will be visited more than once, but if the item
// is stored or deleted concurrently, Range may reflect any mapping for that item.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (s *IntSet) Range(f func(x uint32) bool) {
	sLen := atomic.LoadUint32(&s.len)
	for i := 0; i < int(sLen); i++ {
		item := atomic.LoadUint32(&s.dirty[i])
		if item == 0 {
			continue
		}
		for j := 0; j < platform; j++ {
			if item == 0 {
				break
			}
			if item&1 == 1 {
				if !f(uint32(platform*i + j)) {
					return
				}
			}
			item >>= 1
		}
	}
}

// Len return the number of elements in set
// time complexity: O(N)
func (s *IntSet) Len() int {
	var sum int
	s.Range(func(x uint32) bool {
		sum += 1
		return true
	})
	return sum
}

// Clear remove all elements from the set
// time complexity: O(N/32)
func (s *IntSet) Clear() {
	sLen := int(atomic.LoadUint32(&s.len))
	for i := 0; i < sLen; i++ {
		atomic.StoreUint32(&s.dirty[i], 0)
	}
}

// Copy return a copy of the set
// time complexity: O(N/32)
func (s *IntSet) Copy() *IntSet {
	var n IntSet
	sLen := int(atomic.LoadUint32(&s.len))
	n.dirty = make([]uint32, sLen)
	for i := 0; i < sLen; i++ {
		n.dirty[i] = atomic.LoadUint32(&s.dirty[i])
	}
	return &n
}

// Null report s if an empty set
// time complexity: O(N/32)
func (s *IntSet) Null() bool {
	sLen := int(atomic.LoadUint32(&s.len))
	if sLen == 0 {
		return true
	}
	for i := 0; i < int(atomic.LoadUint32(&s.len)); i++ {
		item := atomic.LoadUint32(&s.dirty[i])
		if item != 0 {
			return false
		}
	}
	return true
}

// Items return all element in the set
// time complexity: O(N)
func (s *IntSet) Items() []uint32 {
	sum := 0
	sLen := atomic.LoadUint32(&s.len)
	array := make([]uint32, sLen*platform)
	s.Range(func(x uint32) bool {
		array = append(array, x)
		sum += 1
		return true
	})
	return array[:sum]
}

// String returns the set as a string of the form "{1 2 3}".
// time complexity: O(N)
func (s *IntSet) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	s.Range(func(x uint32) bool {
		if buf.Len() > len("{") {
			buf.WriteByte(' ')
		}
		fmt.Fprintf(&buf, "%d", x)
		return true
	})
	buf.WriteByte('}')
	return buf.String()
}
