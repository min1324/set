package set

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	// platform bit = 2^setBits,(32/64)
	setBits         = 5 //+ (^uint(0) >> 63)
	platform        = 1 << setBits
	setMesk  uint32 = 1<<setBits - 1
)

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
	max uint32

	// cap(items)
	cap uint32

	// len(items)
	len uint32

	// only increase
	items []uint32
}

const (
	initSize = 1 << 8
)

func (s *IntSet) onceInit(max int) {
	s.once.Do(func() {
		if max < 1 {
			max = initSize
		}
		num := max>>5 + 1
		s.items = make([]uint32, num)
		atomic.StoreUint32(&s.len, uint32(num))
		atomic.StoreUint32(&s.cap, uint32(num))
		atomic.StoreUint32(&s.max, uint32(max))
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
func idxMod(x uint32) (idx, mod int) {
	return int(x >> setBits), int(x & setMesk)
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

// i must < num
func (s *IntSet) load(i int) uint32 {
	return atomic.LoadUint32(&s.items[i])
}

// Cap return queue's cap
func (q *IntSet) Cap() int {
	return int(atomic.LoadUint32(&q.max))
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *IntSet) Load(x uint32) bool {
	if x > s.getMax() {
		return false
	}
	idx, mod := idxMod(x)
	if idx >= int(s.getLen()) {
		// overflow
		return false
	}
	item := s.load(idx)
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
	if x > s.getMax() {
		return false, false
	}
	idx, mod := idxMod(x)

	// verify and grow the items
	if !s.verify(idx) {
		return
	}
	for {
		item := s.load(idx)
		if (item>>mod)&1 == 1 {
			return true, true
		}
		if atomic.CompareAndSwapUint32(&s.items[idx], item, item|(1<<mod)) {
			return false, true
		}
	}
}

func (s *IntSet) verify(idx int) bool {
	for {
		slen := s.getLen()
		if idx < int(slen) {
			return true
		}
		// TODO grow len
		if idx < int(s.getCap()) {
			if casUint32(&s.len, uint32(slen), uint32(idx+1)) {
				return true
			}
		} else {
			return false
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
	if x > s.getMax() {
		return false, false
	}
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
	sNum := uint32(s.getLen())
	for i := 0; i < int(sNum); i++ {
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
	sNum := s.getLen()
	for i := 0; i < int(sNum); i++ {
		atomic.StoreUint32(&s.items[i], 0)
	}
	casUint32(&s.len, sNum, 0)
}

// Copy return a copy of the set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *IntSet) Copy() *IntSet {
	var n IntSet
	n.OnceInit(s.Cap())
	for i := 0; i < int(s.getLen()); i++ {
		n.items[i] = s.load(i)
	}
	return &n
}

// Null report s if an empty set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *IntSet) Null() bool {
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
// worst time complexity: O(32*N)
// best  time complexity: O(N)
func (s *IntSet) Items() []uint32 {
	sum := 0
	sNum := s.getLen()
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

const (
	validBit         = 31
	freezeBit uint32 = 1 << 31

	// the max item can store in set.
	// total has slice: 1<<31, each slice can hold 31 item
	// so maxItem = 1<<31*31
	// but the memory had not enough space.
	// set the maxItem= 1<<24*31
	maxItem uint32 = 1 << 24 * 31

	initCap = 8
)

// SliceSet is a set of non-negative integers.
// Its zero value represents the empty set.
//
// x is an item in set.
// x = (2^setBits)*idx + mod <==> x = 64*idx + mod  or  x = 32*idx + mod
// idx = x/2^setBits (x>>setBits) , mod = x%2^setBits (x&setMask)
// in the set, x is the pesition: dirty[idx]&(1<<mod)
type SliceSet struct {
	once sync.Once
	// can hold the max item
	// only can set in init
	max uint32

	// *entry
	node unsafe.Pointer
}

type node struct {

	// len(data)
	len uint32

	// cap(data)
	cap uint32

	// growing
	resize uint32

	// valid bit:1-31,the 32 bit means evacuted.
	// when evacuting,can't store nor delete.
	data []uint32
}

func (s *SliceSet) init(max int) {
	s.once.Do(func() {
		var n *node
		var cap uint32 = uint32(max>>5 + 1)
		if max < 1 {
			max = int(maxItem)
			cap = initCap
		}
		n = &node{len: 0, cap: cap, data: make([]uint32, cap)}
		atomic.StoreUint32(&s.max, uint32(max))
		atomic.StorePointer(&s.node, unsafe.Pointer(n))
	})
}

func (s *SliceSet) OnceInit(max int) {
	s.init(max)
}

func (s *SliceSet) Max() uint32 {
	return atomic.LoadUint32(&s.max)
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *SliceSet) Load(x uint32) bool {
	if x > s.Max() {
		// overflow
		return false
	}
	idx, mod := idxModS(x)
	n := s.getNode()
	if idx >= int(n.getLen()) {
		return false
	}
	return (s.load(idx)>>mod)&1 == 1
}

// in 64 bit platform
// x = 64*idx + mod
// idx = x/64 (x>>6) , mod = x%64 (x&(1<<6-1))
//
// in 32 bit platform
// x = 32*idx + mod
// idx = x/32 (x>>5) , mod = x%32 (x&(1<<5-1))
func idxModS(x uint32) (idx, mod int) {
	return int(x / 31), int(x % validBit)
}

func (s *SliceSet) getNode() *node {
	n := (*node)(atomic.LoadPointer(&s.node))
	if n == nil {
		s.init(0)
		n = (*node)(atomic.LoadPointer(&s.node))
	}
	return n
}

// i must < num
func (s *SliceSet) load(i int) uint32 {
	return s.getNode().load(i)
}

func (n *node) load(idx int) uint32 {
	return atomic.LoadUint32(&n.data[idx])
}

func (n *node) getLen() uint32 {
	return atomic.LoadUint32(&n.len)
}

// Store adds the non-negative value x to the set.
// return if x overflow
// time complexity: O(1)
func (s *SliceSet) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report if x overflow
// time complexity: O(1)
func (s *SliceSet) LoadOrStore(x uint32) (loaded, ok bool) {
	s.init(0)
	if x > s.Max() {
		// overflow
		return
	}
	idx, mod := idxModS(x)
	if !s.verify(idx) {
		return
	}
	var n *node
	for {
		n = s.getNode()
		item := n.load(idx)
		if (item>>mod)&1 == 1 {
			return true, true
		}
		if n.evacuted(idx) {
			// growing n,need wait.
			runtime.Gosched()
			continue
		}
		if casUint32(&n.data[idx], item, item|(1<<mod)) {
			return false, true
		}
	}
}

func (n *node) evacuted(idx int) bool {
	return atomic.LoadUint32(&n.data[idx])&freezeBit > 0
}

// verify idx if large len,cap
// if idx=cap,need grow
func (s *SliceSet) verify(idx int) bool {
	for {
		n := s.getNode()
		nlen := n.getLen()
		if idx < int(nlen) {
			break
		}
		ncap := n.getCap()
		if idx < int(ncap) {
			if atomic.CompareAndSwapUint32(&n.len, nlen, uint32(idx+1)) {
				break
			}
		}
		if growWork(s, n, uint32(idx+1)) {
			n := s.getNode()
			if idx >= int(n.getCap()) {
				// check if cap overflow
				return false
			}
			break
		}
		runtime.Gosched()
	}
	return true
}

func growWork(s *SliceSet, old *node, cap uint32) bool {
	if !atomic.CompareAndSwapUint32(&old.resize, 0, 1) {
		// other thread growing
		return false
	}
	// caculate new cap
	newCap := old.getCap()
	doubleCap := newCap << 1
	if cap > doubleCap {
		newCap = cap
	} else {
		if newCap < 1024 {
			newCap = doubleCap
		} else {
			// Check 0 < newcap to detect overflow
			// and prevent an infinite loop.
			for 0 < newCap && newCap < cap {
				newCap += newCap / 4
			}
			// Set newcap to the requested cap when
			// the newcap calculation overflowed.
			if newCap <= 0 {
				newCap = cap
			}
		}
	}
	// new node
	nn := &node{
		len:  cap,
		cap:  newCap,
		data: make([]uint32, newCap),
	}
	// evacute old node to new node
	for i := 0; i < int(old.getCap()); i++ {
		// mask the height bit to freezeBit
		item := atomic.AddUint32(&old.data[i], freezeBit)
		nn.store(i, item&^freezeBit)
	}
	ok := atomic.CompareAndSwapPointer(&s.node, unsafe.Pointer(old), unsafe.Pointer(nn))
	if !ok {
		panic("BUG: failed swapping head")
	}
	return true
}

func (n *node) getCap() uint32 {
	return atomic.LoadUint32(&n.cap)
}

func (n *node) store(idx int, val uint32) {
	atomic.StoreUint32(&n.data[idx], val)
}

// Delete remove x from the set
// return if x overflow
// time complexity: O(1)
func (s *SliceSet) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report if x overflow
// time complexity: O(1)
func (s *SliceSet) LoadAndDelete(x uint32) (loaded, ok bool) {
	if x > s.Max() {
		return
	}
	idx, mod := idxModS(x)
	n := s.getNode()
	if idx >= int(n.getLen()) {
		// overflow
		return false, false
	}
	for {
		n = s.getNode()
		item := n.load(idx)
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if n.evacuted(idx) {
			runtime.Gosched()
			continue
		}
		if casUint32(&n.data[idx], item, item&^(1<<mod)) {
			return true, true
		}
	}
}

// Adds add all x in args to the set
// time complexity: O(n)
func (s *SliceSet) Adds(args ...uint32) {
	for _, x := range args {
		s.Store(x)
	}
}

// Removes remove all x in args to the set
// time complexity: O(n)
func (s *SliceSet) Removes(args ...uint32) {
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
func (s *SliceSet) Range(f func(x uint32) bool) {
	n := s.getNode()
	sNum := int(n.getLen())
	for i := 0; i < sNum; i++ {
		item := s.load(i)
		if item == 0 {
			continue
		}
		for j := 0; j < validBit; j++ {
			if item == 0 {
				break
			}
			if item&1 == 1 {
				if !f(uint32(validBit*i + j)) {
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
func (s *SliceSet) Len() int {
	var sum int
	s.Range(func(x uint32) bool {
		sum += 1
		return true
	})
	return sum
}

// Cap return IntSet's max item
func (s *SliceSet) Cap() int {
	return int(s.Max())
}

// Clear remove all elements from the set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *SliceSet) Clear() {
	n := s.getNode()
	sNum := int(n.getLen())
	for i := 0; i < sNum; i++ {
		s.store(i, 0)
	}
}

// i must < num
func (s *SliceSet) store(i int, val uint32) {
	s.verify(i)
	s.getNode().store(i, val)
}

// Copy return a copy of the set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *SliceSet) Copy() *SliceSet {
	var m SliceSet
	n := s.getNode()
	sLen := int(n.getLen())
	for i := 0; i < sLen; i++ {
		m.store(i, n.load(i))
	}
	return &m
}

// Null report s if an empty set
// worst time complexity: O(N)
// best  time complexity: O(1)
func (s *SliceSet) Null() bool {
	m := s.getNode()
	sLen := int(m.getLen())
	if sLen == 0 {
		return true
	}
	for i := 0; i < sLen; i++ {
		if s.load(i) != 0 {
			return false
		}
	}
	return true
}

// Items return all element in the set
// worst time complexity: O(32*N)
// best  time complexity: O(N)
func (s *SliceSet) Items() []uint32 {
	sum := 0
	n := s.getNode()
	sLen := n.getLen()
	array := make([]uint32, 0, sLen*validBit)
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
func (s *SliceSet) String() string {
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

func casUint32(addr *uint32, old, new uint32) bool {
	return atomic.CompareAndSwapUint32(addr, old, new)
}
