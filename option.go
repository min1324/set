package set

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// NewOption15 return a trends set with items args.
// the frist arg is cap,
func NewOption15(cap int, args ...uint32) *Option {
	var s Option
	s.init15(cap)
	Adds(&s, args...)
	return &s
}

// NewOption16 return a static set with items args.
// the frist arg is cap,
func NewOption16(cap int, args ...uint32) *Option {
	var s Option
	s.init16(cap)
	Adds(&s, args...)
	return &s
}

// NewOption31 return a trends set with items args.
// the frist arg is cap,
func NewOption31(cap int, args ...uint32) *Option {
	var s Option
	s.init31(cap)
	Adds(&s, args...)
	return &s
}

// NewOption32 return a static set with items args.
// the frist arg is cap,
func NewOption32(cap int, args ...uint32) *Option {
	var s Option
	s.init32(cap)
	Adds(&s, args...)
	return &s
}

// Option a set can chose trends or static cap
type Option struct {
	once sync.Once

	// max input x
	max uint32

	node unsafe.Pointer
}

func (s *Option) init15(max int) { s.init(max, optEntry15) }
func (s *Option) init16(max int) { s.init(max, optEntry16) }
func (s *Option) init31(max int) { s.init(max, optEntry31) }
func (s *Option) init32(max int) { s.init(max, optEntry32) }

func (s *Option) init(max int, typ optEntryTyp) {
	s.once.Do(func() {
		if max > int(maximum) || max < 1 {
			max = int(maximum)
		}
		e := newOptEntry(uint32(max), typ)
		atomic.StorePointer(&s.node, unsafe.Pointer(e))
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

// OnceInit initialize set use max
// it only execute once time.
// if max<1,init a trends set
// max>1,init a static set
func (s *Option) OnceInit(max int) {
	if max < 1 {
		s.init32(max)
	} else {
		s.init31(max)
	}
}

func (s *Option) getMax() uint32        { return atomic.LoadUint32(&s.max) }
func (s *Option) getLen() uint32        { return s.getEntry().getLen() }
func (s *Option) load(i int) uint32     { return s.getEntry().load(i) }
func (s *Option) store(i int, x uint32) { s.getEntry().store(i, x) }

func (s *Option) getEntry() *entry {
	p := atomic.LoadPointer(&s.node)
	if p == nil {
		s.init15(int(s.getMax()))
		p = atomic.LoadPointer(&s.node)
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
	s.OnceInit(0)
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
func (s *Option) Range(f func(x uint32) bool) {
	e := s.getEntry()
	e.walk(f)
}

type optEntryTyp int

const (
	optEntry32 optEntryTyp = iota
	optEntry31
	optEntry16
	optEntry15
)

type entry struct {
	// needed
	idxMod func(x uint32) (idx, mod uint32)
	typ    optEntryTyp // entry type
	bit    uint32      // item valid bit
	resize uint32      // growing flag
	count  uint32      // number of element in entry
	len    uint32      // len(data)
	cap    uint32      // cap(data)
	data   []uint32    // when evacuting,can't store nor delete.

	// options
	// if trends cap,below two func must not nil
	// if static cap,must nil
	growWork func(s *Option, old *entry, cap uint32) bool
	frozen   func(x uint32) bool
}

// String returns the set as a string of the form "{1 2 3}".
// use for fmt.Print
func (e *entry) String() string {
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

// in 64 bit platform
// x = 64*idx + mod
// idx = x/64 (x>>6) , mod = x%64 (x&(1<<6-1))
//
// in 32 bit platform
// x = idx + mod
// idx = x/32 (x>>5) , mod = x%32 (x&(1<<5-1))
//
// in 16 bit platform
// x = idx + mod
// idx = x/16 (x>>4) , mod = x%16 (x&(1<<4-1))
func optIdxMod32(x uint32) (idx, mod uint32) { return x >> 5, x & 31 }
func optIdxMod31(x uint32) (idx, mod uint32) { return (x / 31), (x % 31) }
func optIdxMod16(x uint32) (idx, mod uint32) { return x >> 4, x & 15 }

func newOptEntry(max uint32, typ optEntryTyp) *entry {
	var nn *entry
	switch typ {
	case optEntry32:
		nn = newOptEntry32(max)
	case optEntry31:
		nn = newOptEntry31(max)
	case optEntry16:
		nn = newOptEntry16(max)
	case optEntry15:
		nn = newOptEntry15(max)
	default:
		nn = newOptEntry32(max)
		// unknown flag
	}
	return nn
}

func newOptEntry32(max uint32) *entry {
	if max < 1 {
		max = initSize
	}
	if max > maximum {
		max = maximum
	}
	cap := max>>5 + 1
	return &entry{cap: cap, bit: 32, data: make([]uint32, cap),
		idxMod: optIdxMod32, typ: optEntry32,
	}
}

func newOptEntry31(max uint32) *entry {
	cap := max/31 + 1
	if max < 1 {
		max = maximum
		cap = initCap
	}
	if max > maximum {
		max = maximum
		cap = max/31 + 1
	}
	return &entry{cap: cap, bit: 31, data: make([]uint32, cap),
		idxMod: optIdxMod31, typ: optEntry31,
		growWork: optGrowWork, frozen: optFrozen,
	}
}

func newOptEntry16(max uint32) *entry {
	if max < 1 {
		max = initSize
	}
	if max > maximum {
		max = maximum
	}
	cap := max>>4 + 1
	return &entry{cap: cap, bit: 16, data: make([]uint32, cap),
		idxMod: optIdxMod16, typ: optEntry16,
	}
}

func newOptEntry15(max uint32) *entry {
	cap := max>>4 + 1
	if max < 1 {
		max = maximum
		cap = initCap
	}
	if max > maximum {
		max = maximum
		cap = max/16 + 1
	}
	return &entry{cap: cap, bit: 16, data: make([]uint32, cap),
		idxMod: optIdxMod16, typ: optEntry15,
		growWork: optGrowWork, frozen: optFrozen,
	}
}

func (e *entry) getMax() uint32 {
	cap := e.getCap()
	bit := atomic.LoadUint32(&e.bit)
	return cap * bit
}

func (e *entry) getLen() uint32   { return atomic.LoadUint32(&e.len) }
func (e *entry) getCap() uint32   { return atomic.LoadUint32(&e.cap) }
func (e *entry) getCount() uint32 { return atomic.LoadUint32(&e.count) }

// load data[i]
func (e *entry) load(i int) uint32 {
	// TODO freebit
	return atomic.LoadUint32(&e.data[i])
}

// store data[i]=val
func (e *entry) store(i int, val uint32) {
	if e.overflow(uint32(i)) {
		return
	}
	atomic.StoreUint32(&e.data[i], val)
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
		if e.frozen != nil {
			if e.frozen(item) {
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
		if e.frozen != nil {
			if e.frozen(item) {
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

func (e *entry) overflow(idx uint32) bool {
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

func optFrozen(x uint32) bool { return x&freezeBit > 0 }

func optGrowWork(s *Option, old *entry, cap uint32) bool {
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
		frozen:   old.frozen,
		typ:      old.typ,
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

func optConvert(s *Option, typ optEntryTyp) bool {
	if s.getEntry().typ == typ {
		return true
	}
	var old *entry
	for {
		old = s.getEntry()
		if atomic.CompareAndSwapUint32(&old.resize, 0, 1) {
			// other thread growing
			break
		}
	}

	// caculate new cap
	max := s.getMax()

	// new node
	nn := newOptEntry(max, typ)
	if nn == nil {
		return false
	}

	for i := 0; i < int(old.getCap()); i++ {
		old.freeze(i)
	}

	optEvacute(old, nn, max)
	nn.walk(func(x uint32) bool {
		nn.count += 1
		return true
	})
	ok := atomic.CompareAndSwapPointer(&s.node, unsafe.Pointer(old), unsafe.Pointer(nn))
	if !ok {
		panic("BUG: failed swapping head")
	}
	return ok
}

func (e *entry) freeze(idx int) uint32 {
	for {
		item := atomic.LoadUint32(&e.data[idx])
		if atomic.CompareAndSwapUint32(&e.data[idx], item, item|freezeBit) {
			return item
		}
	}
}

// ToStatic convert option's entry to entry32
func (s *Option) ToStatic() { optConvert(s, optEntry32) }

// ToStatic convert option's entry to entry31
func (s *Option) ToTrends() { optConvert(s, optEntry31) }

// ToStatic convert option's entry to entry16
func (s *Option) ToStatic16() { optConvert(s, optEntry16) }

// ToStatic convert option's entry to entry15
func (s *Option) ToTrends16() { optConvert(s, optEntry15) }

// optEvacute old entry to new according bit
func optEvacute(old, new *entry, max uint32) {
	oldBit := atomic.LoadUint32(&old.bit)
	newBit := atomic.LoadUint32(&new.bit)
	switch oldBit {
	case 16:
		if newBit == 31 {
			p := newOptEntry32(max)
			u16To32(old, p)
			u32To31(p, new)
		} else {
			u16To32(old, new)
		}
	case 31:
		if newBit == 16 {
			p := newOptEntry32(max)
			u31To32(old, p)
			u32To16(p, new)
		} else {
			u31To32(old, new)
		}
	case 32:
		if newBit == 16 {
			u32To16(old, new)
		} else {
			u32To31(old, new)
		}
	}
	// TODO add other bit
}

func u32To16(old, new *entry) {
	ocap := old.getCap()
	for i := 0; i < int(ocap); i++ {
		item := old.load(i)
		if item == 0 {
			continue
		}
		new.store(2*i, item&(1<<16-1))
		new.store(2*i+1, item>>16)
	}
}

func u16To32(old, new *entry) {
	ocap := old.getCap()
	for i := 0; i < int(ocap); i++ {
		item := old.load(i)
		item &^= freezeBit
		if item == 0 {
			continue
		}
		new.store(i>>1, new.load(i>>1)|item<<(16*(i&1)))
	}
}

func u32To31(old, new *entry) {
	slen := int(old.getLen())
	ncap := int(new.getCap())
	for i := 0; i < slen; i++ {
		item := old.load(i)
		// u32 实际存位值
		ni := i + i/31
		//有效补偿位
		bit := i % 31
		ivalue := item << bit
		// 去掉bit最高位
		ivalue &^= freezeBit
		new.store(ni, new.load(ni)|ivalue)

		if ni+1 < ncap {
			// 补偿i+(i/31+1)
			bv := item >> (31 - bit)
			new.store(ni+1, new.load(ni+1)|bv)
		}
	}
}

func u31To32(old, new *entry) {
	slen := int(old.getLen())
	for i := 0; i < slen; i++ {
		item := old.load(i)
		// 去掉最高位
		item &^= freezeBit
		// u32 实际存位值
		ni := (i + 1) * 31 / 32
		//有效补偿位
		bit := i % 32

		// ni - 补偿
		ivalue := (item &^ (1<<bit - 1)) >> bit
		new.store(ni, new.load(ni)|ivalue)

		// 补偿ni-(i>>5+1)
		if i%32 != 0 {
			bvalue := item & (1<<bit - 1)
			bvalue <<= 31 - ((i - 1) % 32)
			oval := new.load(i - (i>>5 + 1))
			new.store(i-(i>>5+1), oval|bvalue)
		}
	}
}
