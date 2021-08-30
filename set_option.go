package set

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

func NewVarOpt(cap int) *Option {
	var f Option
	f.initVar(cap)
	return &f
}

func NewIntOpt(cap int) *Option {
	var f Option
	f.initInt(cap)
	return &f
}

// Option had an unchange cap entry.
type Option struct {
	once sync.Once

	mu sync.Mutex

	// max input x
	max uint32

	node unsafe.Pointer
}

func (s *Option) onceInit(max int) {
	s.initInt(max)
}

func (s *Option) initInt(max int) {
	s.once.Do(func() {
		if max < 1 {
			max = initSize
		}
		if max > int(maximum) {
			max = int(maximum)
		}
		cap := max>>5 + 1
		e := newIntEntry(uint32(cap))
		atomic.StorePointer(&s.node, unsafe.Pointer(e))
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

func (s *Option) initVar(max int) {
	s.once.Do(func() {
		var cap uint32 = uint32(max>>5 + 1)
		if max < 1 || max > int(maximum) {
			max = int(maximum)
			cap = initCap
		}
		e := newVarEntry(uint32(cap))
		atomic.StorePointer(&s.node, unsafe.Pointer(e))
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

// OnceInit initialize set use max
// it only execute once time.
// if max<1, will use 256.
func (s *Option) OnceInit(max int) {
	s.onceInit(max)
}

func (s *Option) getMax() uint32 {
	return atomic.LoadUint32(&s.max)
}

func (s *Option) getEntry() *entry {
	p := atomic.LoadPointer(&s.node)
	if p == nil {
		s.mu.Lock()
		p = atomic.LoadPointer(&s.node)
		if p == nil {
			s.onceInit(initSize)
			p = atomic.LoadPointer(&s.node)
		}
		s.mu.Unlock()
	}
	return (*entry)(p)
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *Option) Load(x uint32) (ok bool) {
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
func (s *Option) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Option) LoadOrStore(x uint32) (loaded, ok bool) {
	s.onceInit(0)
	if x > s.getMax() {
		// overflow
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
		if e.growWork == nil {
			// not need grow
			return false, false
		}
		e.growWork(s, e, idx+1)
	}
}

// Delete remove x from the set
// return true if success, false if x overflow
// time complexity: O(1)
func (s *Option) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Option) LoadAndDelete(x uint32) (loaded, ok bool) {
	if x > s.getMax() {
		return false, false
	}
	for {
		e := s.getEntry()
		idx, mod := e.idxMod(x)
		if idx >= e.getLen() {
			return false, false
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
func (s *Option) Range(f func(x uint32) bool) {
	e := s.getEntry()
	e.walk(f)
}

// Cap return IntSet's max item can store
func (s *Option) Cap() int {
	return int(atomic.LoadUint32(&s.max))
}

// Len return the number of elements in set
// worst time complexity: worst: O(N)
// best  time complexity: O(1)
func (s *Option) Len() int {
	e := s.getEntry()
	count := e.getCount()
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
	atomic.CompareAndSwapUint32(&e.count, 0, uint32(sum))
	return sum
}

// Clear remove all elements from the set
// time complexity: O(N/32)
func (s *Option) Clear() {
	e := s.getEntry()
	sLen := e.getLen()
	for i := 0; i < int(sLen); i++ {
		e.store(i, 0)
	}
	atomic.StoreUint32(&e.count, 0)
	atomic.CompareAndSwapUint32(&e.len, sLen, 0)
}

// Copy return a copy of the set
// time complexity: O(N)
func (s *Option) Copy() *IntSet {
	var n IntSet
	n.OnceInit(s.Cap())
	e := s.getEntry()
	slen := e.getLen()
	n.len = slen
	for i := 0; i < int(slen); i++ {
		n.items[i] = e.load(i)
	}
	// update count
	n.Len()
	return &n
}

// Null report set if empty
// time complexity: O(N/32)
func (s *Option) Null() bool {
	e := s.getEntry()
	if e.getCount() != 0 {
		return false
	}
	if e.getLen() == 0 {
		return true
	}
	for i := 0; i < int(e.getLen()); i++ {
		item := e.load(i)
		if item != 0 {
			return false
		}
	}
	return true
}

// Items return all element in the set
// time complexity: O(N)
func (s *Option) Items() []uint32 {
	e := s.getEntry()
	sLen := e.getLen()
	array := make([]uint32, 0, sLen*platform)
	sum := 0
	s.Range(func(x uint32) bool {
		array = append(array, x)
		sum += 1
		return true
	})
	return array[:sum]
}

// String returns the set as a string of the form "{1 2 3}".
// time complexity: O(N)
func (s *Option) String() string {
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

type entry struct {
	// needed
	idxMod func(x uint32) (idx, mod uint32)

	// options
	growWork func(s *Option, old *entry, cap uint32) bool
	evacuted func(e *entry, idx uint32) bool

	// growing
	resize uint32

	bit uint32

	count uint32

	// len(data)
	len uint32

	// cap(data)
	cap uint32

	// valid bit:0-31,the 32 bit means evacuted.
	// when evacuting,can't store nor delete.
	data []uint32
}

func newIntEntry(cap uint32) *entry {
	return &entry{cap: cap, bit: 32, data: make([]uint32, cap), idxMod: intIdxMod}
}

// in 64 bit platform
// x = 64*idx + mod
// idx = x/64 (x>>6) , mod = x%64 (x&(1<<6-1))
//
// in 32 bit platform
// x = idx + mod
// idx = x/32 (x>>5) , mod = x%32 (x&(1<<5-1))
func intIdxMod(x uint32) (idx, mod uint32) {
	return x >> setBits, x & setMask
}

func newVarEntry(cap uint32) *entry {
	return &entry{cap: cap, bit: 31, data: make([]uint32, cap), idxMod: varIdxMod,
		growWork: entryGrowWork, evacuted: evacuted,
	}
}

func varIdxMod(x uint32) (idx, mod uint32) {
	return (x / 31), (x % 31)
}

func (e *entry) getCount() uint32 {
	return atomic.LoadUint32(&e.count)
}

func (e *entry) getLen() uint32 {
	return atomic.LoadUint32(&e.len)
}

func (e *entry) getCap() uint32 {
	return atomic.LoadUint32(&e.cap)
}

func (e *entry) load(i int) uint32 {
	return atomic.LoadUint32(&e.data[i])
}

func (e *entry) store(i int, val uint32) {
	atomic.StoreUint32(&e.data[i], val)
}

func (e *entry) cas(i int, old, new uint32) bool {
	return atomic.CompareAndSwapUint32(&e.data[i], old, new)
}

func (e *entry) tryLoad(idx, mod uint32) (ok bool) {
	item := atomic.LoadUint32(&e.data[idx])
	return (item>>mod)&1 == 1
}

func (e *entry) tryStore(idx, mod uint32) (loaded, ok bool) {
	for {
		item := atomic.LoadUint32(&e.data[idx])
		if (item>>mod)&1 == 1 {
			// already in set
			return true, true
		}
		if e.evacuted != nil {
			if e.evacuted(e, idx) {
				return false, false
			}
		}
		if atomic.CompareAndSwapUint32(&e.data[idx], item, item|(1<<mod)) {
			atomic.AddUint32(&e.count, 1)
			return false, true
		}
	}
}

func (e *entry) tryDelete(idx, mod uint32) (loaded, ok bool) {
	for {
		item := atomic.LoadUint32(&e.data[idx])
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if e.evacuted != nil {
			if e.evacuted(e, idx) {
				return false, false
			}
		}
		if atomic.CompareAndSwapUint32(&e.data[idx], item, item&^(1<<mod)) {
			atomic.AddUint32(&e.count, ^uint32(0))
			return true, true
		}
	}
}

func (e *entry) walk(f func(x uint32) bool) {
	sLen := e.getLen()
	bit := int(atomic.LoadUint32(&e.bit))
	for i := 0; i < int(sLen); i++ {
		item := e.load(i)
		if item == 0 {
			continue
		}
		for j := 0; j < bit; j++ {
			if item == 0 {
				break
			}
			if item&1 == 1 {
				if !f(uint32(bit*i + j)) {
					return
				}
			}
			item >>= 1
		}
	}
}

func evacuted(e *entry, idx uint32) bool {
	return atomic.LoadUint32(&e.data[idx])&freezeBit > 0
}

func (e *entry) overflow(idx uint32) bool {
	for {
		slen := e.getLen()
		if idx < slen {
			return false
		}
		// idx > len
		// TODO grow len to idx+1
		if idx >= e.getCap() {
			// idx > cap, overflow
			return true
		}
		if atomic.CompareAndSwapUint32(&e.len, uint32(slen), uint32(idx+1)) {
			return false
		}
	}
}

func entryGrowWork(s *Option, old *entry, cap uint32) bool {
	if !atomic.CompareAndSwapUint32(&old.resize, 0, 1) {
		// other thread growing
		return false
	}
	// caculate new cap
	newCap := caculateCap(old.getCap(), cap)

	// new node
	nn := &entry{
		idxMod:   old.idxMod,
		growWork: old.growWork,
		evacuted: old.evacuted,
		bit:      atomic.LoadUint32(&old.bit),
		len:      cap,
		cap:      newCap,
		data:     make([]uint32, newCap),
	}
	// evacute old node to new node
	for i := 0; i < int(old.getCap()); i++ {
		// mask the height bit to freezeBit
		item := atomic.AddUint32(&old.data[i], freezeBit)
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
