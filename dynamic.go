package set

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Dynamic return a dynamic set with max.
// set has range [0,max],max default 256.
// each group of data has 16 item,
// general use for heigh concurrent but item range not large.
type Dynamic struct {
	once sync.Once

	// max input x
	max uint32

	// *baseEntry
	node unsafe.Pointer
}

func (s *Dynamic) init(max int) {
	s.once.Do(func() {
		e := newNode(uint32(max))
		atomic.StorePointer(&s.node, unsafe.Pointer(e))
		if max < 1 {
			max = int(maximum)
		}
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

func (s *Dynamic) onceInit(max int) { s.init(max) }

// OnceInit initialize set use max
// it only execute once time.
// if max<1,init a trends set
// max>1,init a static set
func (s *Dynamic) OnceInit(max int) { s.init(max) }

func (s *Dynamic) getMax() uint32    { return atomic.LoadUint32(&s.max) }
func (s *Dynamic) getCap() uint32    { return s.getEntry().getCap() }
func (s *Dynamic) getLen() uint32    { return s.getEntry().getLen() }
func (s *Dynamic) load(i int) uint32 { return s.getEntry().load(i) }

func (s *Dynamic) store(i int, x uint32) {
	s.getEntry().store(i, x)
}

func (s *Dynamic) getEntry() *dynEntry {
	p := atomic.LoadPointer(&s.node)
	if p == nil {
		s.init(int(s.getMax()))
		p = atomic.LoadPointer(&s.node)
	}
	return (*dynEntry)(p)
}

func (e *dynEntry) idxMod(x uint32) (idx, mod uint32) { return x >> 4, x & 15 }

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *Dynamic) Load(x uint32) (ok bool) {
	if x > s.getMax() {
		// overflow
		return false
	}
	e := s.getEntry()
	if e == nil {
		return false
	}
	idx, mod := e.idxMod(x)
	if idx >= e.getLen() {
		return false
	}
	return e.tryLoad(idx, mod)
}

// Store adds the non-negative value x to the set.
// return false if x overflow bigger than max (default 256).
// time complexity: O(1)
func (s *Dynamic) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Dynamic) LoadOrStore(x uint32) (loaded, ok bool) {
	s.OnceInit(0)
	if x > s.getMax() {
		return false, false
	}
	for {
		e := s.getEntry()
		idx, mod := e.idxMod(x)
		if !e.overflow(idx) {
			loaded, ok = e.tryStore(idx, mod)
			if ok {
				return loaded, true
			}
		}
		dynGrowWork(s, e, idx+1)
	}
}

// Delete remove x from the set
// return true if success, false if x overflow
// time complexity: O(1)
func (s *Dynamic) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Dynamic) LoadAndDelete(x uint32) (loaded, ok bool) {
	if x > s.getMax() {
		return false, false
	}
	for {
		e := s.getEntry()
		idx, mod := e.idxMod(x)
		if idx >= e.getLen() {
			return false, true
		}
		loaded, ok = e.tryDelete(idx, mod)
		if ok {
			return loaded, true
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
// example set:
// {31,63,...,n-1}
func (s *Dynamic) Range(f func(x uint32) bool) {
	e := s.getEntry()
	e.walk(f)
}

type dynEntry struct {
	resize uint32
	count  uint32   // number of element in dynEntry
	len    uint32   // len(data)
	cap    uint32   // cap(data)
	data   []uint32 // when evacuting,can't store nor delete.
}

// String returns the set as a string of the form "{1 2 3}".
// use for fmt.Print
func (e *dynEntry) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	e.walk(func(x uint32) bool {
		if buf.Len() > len("{") {
			buf.WriteByte(' ')
		}
		fmt.Fprintf(&buf, "%d", x)
		return true
	})
	buf.WriteByte('}')
	return buf.String()
}

func newNode(max uint32) *dynEntry {
	cap := max>>4 + 1
	if max < 1 {
		max = maximum
		cap = initCap
	}
	return &dynEntry{cap: cap, data: make([]uint32, cap)}
}

func (e *dynEntry) getMax() uint32 { return e.getCap() << 4 }
func (e *dynEntry) getLen() uint32 { return atomic.LoadUint32(&e.len) }
func (e *dynEntry) getCap() uint32 { return atomic.LoadUint32(&e.cap) }

// load data[i]
func (e *dynEntry) load(i int) uint32 { return atomic.LoadUint32(&e.data[i]) }

// store data[i]=val
func (e *dynEntry) store(i int, val uint32) {
	if e.overflow(uint32(i)) {
		return
	}
	atomic.StoreUint32(&e.data[i], val)
}

func (e *dynEntry) tryLoad(idx, mod uint32) (ok bool) {
	item := atomic.LoadUint32(&e.data[idx])
	return (item>>mod)&1 == 1
}

func (e *dynEntry) tryStore(idx, mod uint32) (loaded, ok bool) {
	for {
		item := atomic.LoadUint32(&e.data[idx])
		if (item>>mod)&1 == 1 {
			return true, true
		}
		if atomic.CompareAndSwapUint32(&e.data[idx], item, item|(1<<mod)) {
			atomic.AddUint32(&e.count, 1)
			return false, true
		}
	}
}

func (e *dynEntry) tryDelete(idx, mod uint32) (loaded, ok bool) {
	for {
		item := atomic.LoadUint32(&e.data[idx])
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if atomic.CompareAndSwapUint32(&e.data[idx], item, item&^(1<<mod)) {
			atomic.AddUint32(&e.count, ^uint32(0))
			return true, true
		}
	}
}

func (e *dynEntry) walk(f func(x uint32) bool) {
	sLen := e.getLen()
	for i := 0; i < int(sLen); i++ {
		item := e.load(i)
		if item == 0 {
			continue
		}
		// TODO add to valid bit
		for j := 0; j < 16; j++ {
			if item == 0 {
				break
			}
			if item&1 == 1 {
				if !f(uint32(i<<4 + j)) {
					return
				}
			}
			item >>= 1
		}
	}
}

func (e *dynEntry) overflow(idx uint32) bool {
	for {
		slen := e.getLen()
		if idx < slen {
			return false
		}
		// idx > len
		if idx >= e.getCap() {
			// idx > cap, overflow
			return true
		}
		if atomic.CompareAndSwapUint32(&e.len, uint32(slen), uint32(idx+1)) {
			return false
		}
	}
}

func dynGrowWork(s *Dynamic, old *dynEntry, cap uint32) bool {
	if !atomic.CompareAndSwapUint32(&old.resize, 0, 1) {
		// other thread growing
		return false
	}
	// caculate new cap
	newCap := caculateCap(old.getCap(), cap)

	// new node
	nn := &dynEntry{
		len:  cap,
		cap:  newCap,
		data: make([]uint32, newCap),
	}
	// evacute old node to new node
	for i := 0; i < int(old.getCap()); i++ {
		// mask the height bit to freezeBit
		item := old.freeze(i)
		nn.store(i, item&^freezeBit)
	}
	nn.walk(func(x uint32) bool {
		nn.count += 1
		return true
	})
	ok := atomic.CompareAndSwapPointer(&s.node, unsafe.Pointer(old), unsafe.Pointer(nn))
	if !ok {
		panic("BUG: failed swapping head")
	}
	return true
}

func (e *dynEntry) freeze(idx int) uint32 {
	for {
		item := atomic.LoadUint32(&e.data[idx])
		if atomic.CompareAndSwapUint32(&e.data[idx], item, item|freezeBit) {
			return item
		}
	}
}
