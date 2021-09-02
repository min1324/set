package set

import (
	"sync"
	"sync/atomic"
)

// Static a set of non-negative integers.
// Its zero value represents the empty set.
//
// once init, set cap not change any more
//
// x is an item in set.
// x = (2^setBits)*idx + mod <==> x = 64*idx + mod  or  x = idx + mod
// so that:idx = x/2^setBits (x>>setBits), mod = x%2^setBits (x&setMesk)
// in the set, x is the pesition: dirty[idx]&(1<<mod)
type Static struct {
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

	data []uint32
}

func (s *Static) onceInit(max int) {
	s.once.Do(func() {
		if max < 1 {
			max = initSize
		}
		if max > int(maximum) {
			max = int(maximum)
		}
		num := max>>5 + 1
		s.data = make([]uint32, num)
		atomic.StoreUint32(&s.cap, uint32(num))
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

// OnceInit initialize set use max
// it only execute once time.
// if max<1, will use 256.
func (s *Static) OnceInit(max int) { s.onceInit(max) }

// Init initialize IntSet use default max: 256
// it only execute once time.
func (s *Static) Init() { s.onceInit(initSize) }

func (s *Static) getLen() uint32    { return atomic.LoadUint32(&s.len) }
func (s *Static) getCap() uint32    { return atomic.LoadUint32(&s.cap) }
func (s *Static) getMax() uint32    { return atomic.LoadUint32(&s.max) }
func (s *Static) load(i int) uint32 { return atomic.LoadUint32(&s.data[i]) }

func (s *Static) store(i int, x uint32) {
	if s.overflow(i) {
		return
	}
	atomic.StoreUint32(&s.data[i], x)
}

// in 64 bit platform
// x = 64*idx + mod
// idx = x/64 (x>>6) , mod = x%64 (x&(1<<6-1))
//
// in 32 bit platform
// x = idx + mod
// idx = x/32 (x>>5) , mod = x%32 (x&(1<<5-1))
func (s *Static) idxMod(x uint32) (idx, mod int) {
	return int(x >> 5), int(x & 31)
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *Static) Load(x uint32) bool {
	if x > s.getMax() {
		// overflow
		return false
	}
	idx, mod := s.idxMod(x)
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
func (s *Static) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Static) LoadOrStore(x uint32) (loaded, ok bool) {
	s.onceInit(initSize)
	if x > s.getMax() {
		// overflow
		return false, false
	}
	idx, mod := s.idxMod(x)
	// verify the idx
	if s.overflow(idx) {
		return false, false
	}
	for {
		item := s.load(idx)
		if (item>>mod)&1 == 1 {
			// already in set
			return true, true
		}
		if atomic.CompareAndSwapUint32(&s.data[idx], item, item|(1<<mod)) {
			atomic.AddUint32(&s.count, 1)
			return false, true
		}
	}
}

// Delete remove x from the set
// return true if success, false if x overflow
// time complexity: O(1)
func (s *Static) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Static) LoadAndDelete(x uint32) (loaded, ok bool) {
	if x > s.getMax() {
		// overflow
		return false, false
	}
	s.onceInit(initSize)
	idx, mod := s.idxMod(x)
	if idx >= int(s.getLen()) {
		// not in set
		return false, true
	}
	for {
		item := s.load(idx)
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if atomic.CompareAndSwapUint32(&s.data[idx], item, item&^(1<<mod)) {
			atomic.AddUint32(&s.count, ^uint32(0))
			return true, true
		}
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
// example set: {31,63,...,32*n-1}
// this will case O(max),max give in init.
func (s *Static) Range(f func(x uint32) bool) {
	sLen := uint32(s.getLen())
	for i := 0; i < int(sLen); i++ {
		item := s.load(i)
		if item == 0 {
			continue
		}
		for j := 0; j < 32; j++ {
			if item == 0 {
				break
			}
			if item&1 == 1 {
				if !f(uint32(32*i + j)) {
					return
				}
			}
			item >>= 1
		}
	}
}

// overflow update current len to idx+1 if idx>len
// return false if idx>cap
func (s *Static) overflow(idx int) bool {
	for {
		slen := s.getLen()
		if idx < int(slen) {
			return false
		}
		// idx > len,grow len to idx+1
		if idx >= int(s.getCap()) {
			// idx > cap, overflow
			return true
		}
		if atomic.CompareAndSwapUint32(&s.len, uint32(slen), uint32(idx+1)) {
			return false
		}
	}
}
