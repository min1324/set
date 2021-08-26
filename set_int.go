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

// New return a set with items args.
// cap is set cap,if cap<1,will use 256.
func New(cap int, args ...uint32) *IntSet {
	var s IntSet
	s.OnceInit(cap)
	s.Adds(args...)
	return &s
}

// IntSet is a set of non-negative integers.
// Its zero value represents the empty set.
//
// x is an item in set.
// x = (2^setBits)*idx + mod <==> x = 64*idx + mod  or  x = 32*idx + mod
// idx = x/2^setBits (x>>setBits) , mod = x%2^setBits (x&setMesk)
// in the set, x is the pesition: dirty[idx]&(1<<mod)
type IntSet struct {
	once sync.Once

	// max input x
	cap uint32

	// len(items)
	num uint32

	// only increase
	items []uint32
}

const (
	initSize = 1 << 8
)

func (s *IntSet) onceInit(cap int) {
	s.once.Do(func() {
		if cap < 1 {
			cap = initSize
		}
		num := cap>>5 + 1
		s.items = make([]uint32, num)
		atomic.StoreUint32(&s.num, uint32(num))
		atomic.StoreUint32(&s.cap, uint32(cap))
	})
}

// OnceInit initialize set use cap
// it only execute once time.
// if cap<1, will use 256.
func (s *IntSet) OnceInit(cap int) {
	s.onceInit(cap)
}

// Init initialize queue use default size: 256
// it only execute once time.
func (s *IntSet) Init() {
	s.OnceInit(0)
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

func (s *IntSet) maxIndex() int {
	return int(atomic.LoadUint32(&s.num))
}

// i must < num
func (s *IntSet) loadIdx(i uint32) uint32 {
	return atomic.LoadUint32(&s.items[i])
}

// Cap return queue's cap
func (q *IntSet) Cap() int {
	return int(atomic.LoadUint32(&q.cap))
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *IntSet) Load(x uint32) bool {
	if x > atomic.LoadUint32(&s.cap) {
		return false
	}
	idx, mod := idxMod(x)
	if idx >= uint32(s.maxIndex()) {
		// overflow
		return false
	}
	item := s.loadIdx(idx)
	// return s.dirty[idx]&(1<<mod) != 0
	return (item>>mod)&1 == 1
}

// Store adds the non-negative value x to the set.
// return if x overflow
// time complexity: O(1)
func (s *IntSet) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report if x overflow
// time complexity: O(1)
func (s *IntSet) LoadOrStore(x uint32) (loaded, ok bool) {
	s.Init()
	if x > atomic.LoadUint32(&s.cap) {
		return false, false
	}
	idx, mod := idxMod(x)
	if idx >= uint32(s.maxIndex()) {
		return false, false
	}

	for {
		item := s.loadIdx(idx)
		if (item>>mod)&1 == 1 {
			return true, true
		}
		if atomic.CompareAndSwapUint32(&s.items[idx], item, item|(1<<mod)) {
			return false, true
		}
	}
}

// Delete remove x from the set
// return if x overflow
// time complexity: O(1)
func (s *IntSet) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report if x overflow
// time complexity: O(1)
func (s *IntSet) LoadAndDelete(x uint32) (loaded, ok bool) {
	s.Init()
	if x > atomic.LoadUint32(&s.cap) {
		return false, false
	}
	idx, mod := idxMod(x)
	if idx >= uint32(s.maxIndex()) {
		// overflow
		return false, false
	}
	for {
		item := s.loadIdx(idx)
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if atomic.CompareAndSwapUint32(&s.items[idx], item, item&^(1<<mod)) {
			return true, true
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
// Range may be O(32*N) with the worst time complexity.
// example set:
// {31,63,...,32*n-1}
//
// worst time complexity: O(32*N)
// best  time complexity: O(N)
func (s *IntSet) Range(f func(x uint32) bool) {
	sNum := uint32(s.maxIndex())
	for i := 0; i < int(sNum); i++ {
		item := s.loadIdx(uint32(i))
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
// worst time complexity: worst: O(32*N)
// best  time complexity: O(N)
func (s *IntSet) Len() int {
	var sum int
	s.Range(func(x uint32) bool {
		sum += 1
		return true
	})
	return sum
}

// Clear remove all elements from the set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *IntSet) Clear() {
	sNum := s.maxIndex()
	for i := 0; i < sNum; i++ {
		atomic.StoreUint32(&s.items[i], 0)
	}
}

// Copy return a copy of the set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *IntSet) Copy() *IntSet {
	var n IntSet
	n.OnceInit(s.Cap())
	for i := 0; i < s.maxIndex(); i++ {
		n.items[i] = s.loadIdx(uint32(i))
	}
	return &n
}

// Null report s if an empty set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *IntSet) Null() bool {
	if s.maxIndex() == 0 {
		return true
	}
	for i := 0; i < s.maxIndex(); i++ {
		item := s.loadIdx(uint32(i))
		if item != 0 {
			return false
		}
	}
	return true
}

// Items return all element in the set
// worst time complexity: O(32*N)
// best  time complexity: O(N)
func (s *IntSet) Items() []uint32 {
	sum := 0
	sNum := s.maxIndex()
	array := make([]uint32, 0, sNum*platform)
	s.Range(func(x uint32) bool {
		array = append(array, x)
		sum += 1
		return true
	})
	return array[:sum]
}

// String returns the set as a string of the form "{1 2 3}".
// worst time complexity: O(32*N)
// best  time complexity: O(N)
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
