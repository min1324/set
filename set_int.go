package set

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	// x>>setBits <==> x/(1<<setBits)
	// in 32 platform,setBits=5
	// in 64 platform,setBits=6
	setBits uint32 = 5 //+ (^uint(0) >> 63)

	// x&setMask <==> x%(1<<setMask)
	// in 32 platform,setMask=31
	// in 64 platform,setMask=63
	setMask uint32 = 1<<setBits - 1

	// platform bit = 2^setBits,(32/64)
	platform = 1 << setBits

	initSize = 1 << 8
)

// IntSet is a set of non-negative integers.
// Its zero value represents the empty set.
//
// x is an item in set.
// x = (2^setBits)*idx + mod <==> x = 64*idx + mod  or  x = idx + mod
// so that:idx = x/2^setBits (x>>setBits), mod = x%2^setBits (x&setMesk)
// in the set, x is the pesition: dirty[idx]&(1<<mod)
type IntSet struct {
	once sync.Once

	// max input x
	max uint32

	// number of item in set
	// if count==0,set may be create by public op.
	count uint32

	// cap(items),prevent data race
	cap uint32

	// len(items),idx cursor
	len uint32

	items []uint32
}

func (s *IntSet) onceInit(max int) {
	s.once.Do(func() {
		if max < 1 {
			max = initSize
		}
		if max > int(maximum) {
			max = int(maximum)
		}
		num := max>>5 + 1
		s.items = make([]uint32, num)
		atomic.StoreUint32(&s.cap, uint32(num))
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

// OnceInit initialize set use max
// it only execute once time.
// if max<1, will use 256.
func (s *IntSet) OnceInit(max int) {
	s.onceInit(max)
}

// Init initialize IntSet use default max: 256
// it only execute once time.
func (s *IntSet) Init() {
	s.onceInit(initSize)
}

// in 64 bit platform
// x = 64*idx + mod
// idx = x/64 (x>>6) , mod = x%64 (x&(1<<6-1))
//
// in 32 bit platform
// x = idx + mod
// idx = x/32 (x>>5) , mod = x%32 (x&(1<<5-1))
func idxMod(x uint32) (idx, mod int) {
	return int(x >> setBits), int(x & setMask)
}

func (s *IntSet) getLen() uint32 {
	return atomic.LoadUint32(&s.len)
}

func (s *IntSet) getCap() uint32 {
	return atomic.LoadUint32(&s.cap)
}

func (s *IntSet) getMax() uint32 {
	return atomic.LoadUint32(&s.max)
}

// i must < len
func (s *IntSet) load(i int) uint32 {
	return atomic.LoadUint32(&s.items[i])
}

// i must < len
func (s *IntSet) store(i int, x uint32) {
	s.overflow(i)
	atomic.StoreUint32(&s.items[i], x)
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *IntSet) Load(x uint32) bool {
	if x > s.getMax() {
		// overflow
		return false
	}
	idx, mod := idxMod(x)
	if idx >= int(s.getLen()) {
		// not in set
		return false
	}
	item := s.load(idx)
	return (item>>mod)&1 == 1
}

// Store adds the non-negative value x to the set.
// return false if x overflow bigger than max (default 256).
// time complexity: O(1)
func (s *IntSet) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *IntSet) LoadOrStore(x uint32) (loaded, ok bool) {
	s.onceInit(initSize)
	if x > s.getMax() {
		// overflow
		return false, false
	}
	idx, mod := idxMod(x)

	// verify and grow the items
	if s.overflow(idx) {
		return
	}
	for {
		item := s.load(idx)
		if (item>>mod)&1 == 1 {
			// already in set
			return true, true
		}
		if atomic.CompareAndSwapUint32(&s.items[idx], item, item|(1<<mod)) {
			atomic.AddUint32(&s.count, 1)
			return false, true
		}
	}
}

func (s *IntSet) overflow(idx int) bool {
	for {
		slen := s.getLen()
		if idx < int(slen) {
			return false
		}
		// idx > len
		// TODO grow len to idx+1
		if idx >= int(s.getCap()) {
			// idx > cap, overflow
			return true
		}
		if atomic.CompareAndSwapUint32(&s.len, uint32(slen), uint32(idx+1)) {
			return false
		}
	}
}

// Delete remove x from the set
// return true if success, false if x overflow
// time complexity: O(1)
func (s *IntSet) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *IntSet) LoadAndDelete(x uint32) (loaded, ok bool) {
	if x > s.getMax() {
		return false, false
	}
	s.onceInit(initSize)
	idx, mod := idxMod(x)
	if idx >= int(s.getLen()) {
		// overflow
		return false, false
	}
	for {
		item := s.load(idx)
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if atomic.CompareAndSwapUint32(&s.items[idx], item, item&^(1<<mod)) {
			atomic.AddUint32(&s.count, ^uint32(0))
			return true, true
		}
	}
}

// Adds Store all x in args to the set
// ignore x if overflow
// time complexity: O(n)
func (s *IntSet) Adds(args ...uint32) {
	for _, x := range args {
		s.Store(x)
	}
}

// Removes Delete all x in args from the set
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
// Range may be O(N) with the worst time complexity.
// example set:
// {31,63,...,n-1}
func (s *IntSet) Range(f func(x uint32) bool) {
	sLen := uint32(s.getLen())
	for i := 0; i < int(sLen); i++ {
		item := s.load(i)
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

// Cap return IntSet's max item can store
func (s *IntSet) Cap() int {
	return int(atomic.LoadUint32(&s.max))
}

// Len return the number of elements in set
// worst time complexity: worst: O(N)
// best  time complexity: O(1)
func (s *IntSet) Len() int {
	count := atomic.LoadUint32(&s.count)
	if count != 0 {
		return int(count)
	}
	// set may be new with public op.
	// check again with range
	var sum int
	s.Range(func(x uint32) bool {
		sum += 1
		return true
	})
	// try update len
	atomic.CompareAndSwapUint32(&s.count, 0, uint32(sum))
	return sum
}

// Clear remove all elements from the set
// time complexity: O(N/32)
func (s *IntSet) Clear() {
	sLen := s.getLen()
	for i := 0; i < int(sLen); i++ {
		atomic.StoreUint32(&s.items[i], 0)
	}
	atomic.StoreUint32(&s.count, 0)
	atomic.CompareAndSwapUint32(&s.len, sLen, 0)
}

// Copy return a copy of the set
// time complexity: O(N)
func (s *IntSet) Copy() *IntSet {
	var n IntSet
	n.OnceInit(s.Cap())
	slen := s.getLen()
	n.len = slen
	for i := 0; i < int(slen); i++ {
		n.items[i] = s.load(i)
	}
	// update count
	n.Len()
	return &n
}

// Null report set if empty
// time complexity: O(N/32)
func (s *IntSet) Null() bool {
	if atomic.LoadUint32(&s.count) != 0 {
		return false
	}
	if s.getLen() == 0 {
		return true
	}
	for i := 0; i < int(s.getLen()); i++ {
		item := s.load(i)
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
	sLen := s.getLen()
	array := make([]uint32, 0, sLen*platform)
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
