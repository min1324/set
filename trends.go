package set

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	validBit         = 31
	initCap          = 8
	freezeBit uint32 = 1 << 31
)

// Trends is a set of non-negative integers.
// Its zero value represents the empty set.
type Trends struct {
	once sync.Once
	// can hold the max item
	// only can set in init
	max uint32

	// number of item in set
	// if count==0,set may be create by public op.
	count uint32

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

	// valid bit:0-31,the 32 bit means evacuted.
	// when evacuting,can't store nor delete.
	data []uint32
}

func (s *Trends) onceInit(max int) {
	s.once.Do(func() {
		n := newNode(max)
		atomic.StorePointer(&s.node, unsafe.Pointer(n))
		if max < 1 {
			max = int(maximum)
		}
		if max > int(maximum) {
			max = int(maximum)
		}
		atomic.StoreUint32(&s.max, uint32(max))
	})
}

// OnceInit initialize set use max
// it only execute once time.
// if max<1, will use maximum.
func (s *Trends) OnceInit(max int) { s.onceInit(max) }

// Init initialize IntSet use maximum
// it only execute once time.
func (s *Trends) Init() { s.onceInit(0) }

// due to the heightest bit use for evacuted.
// x = 31*idx+mod
// idx = x/31, mod = x%31
func idxMod31(x uint32) (idx, mod int) {
	return int(x / 31), int(x % validBit)
}

func newNode(max int) *node {
	var cap uint32 = uint32(max/31 + 1)
	if max < 1 || max > int(maximum) {
		max = int(maximum)
		cap = initCap
	}
	return &node{len: 0, cap: cap, data: make([]uint32, cap)}
}

func (n *node) getLen() uint32   { return atomic.LoadUint32(&n.len) }
func (n *node) getCap() uint32   { return atomic.LoadUint32(&n.cap) }
func (s *Trends) getMax() uint32 { return atomic.LoadUint32(&s.max) }
func (s *Trends) getLen() uint32 { return s.getNode().getLen() }
func (s *Trends) getCap() uint32 { return s.getNode().getCap() }

func (n *node) load(idx int) uint32       { return atomic.LoadUint32(&n.data[idx]) }
func (n *node) store(idx int, val uint32) { atomic.StoreUint32(&n.data[idx], val) }

// i must < num
func (s *Trends) store(i int, val uint32) {
	s.overflow(i)
	s.getNode().store(i, val)
}

// i must < num
func (s *Trends) load(i int) uint32 {
	n := s.getNode()
	return n.load(i)
}

func (s *Trends) getNode() *node {
	n := (*node)(atomic.LoadPointer(&s.node))
	if n == nil {
		s.Init()
		n = (*node)(atomic.LoadPointer(&s.node))
	}
	return n
}

// Load reports whether the set contains the non-negative value x.
// time complexity: O(1)
func (s *Trends) Load(x uint32) bool {
	if x > s.getMax() {
		// overflow
		return false
	}
	idx, mod := idxMod31(x)
	n := s.getNode()
	if idx >= int(n.getLen()) {
		// not in set
		return false
	}
	item := s.load(idx)
	return (item>>mod)&1 == 1
}

// Store adds the non-negative value x to the set.
// return if x overflow
// time complexity: O(1)
func (s *Trends) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

// LoadOrStore adds the non-negative value x to the set.
// loaded report x if in set,ok report if x overflow
// time complexity: O(1)
func (s *Trends) LoadOrStore(x uint32) (loaded, ok bool) {
	s.onceInit(0)
	if x > s.getMax() {
		// overflow
		return false, false
	}
	idx, mod := idxMod31(x)
	if s.overflow(idx) {
		return false, true
	}
	for {
		n := s.getNode()
		item := n.load(idx)
		if (item>>mod)&1 == 1 {
			// already in set
			return true, true
		}
		if n.evacuted(item) {
			// growing n,need wait.
			runtime.Gosched()
			continue
		}
		if atomic.CompareAndSwapUint32(&n.data[idx], item, item|(1<<mod)) {
			atomic.AddUint32(&s.count, 1)
			return false, true
		}
	}
}

// Delete remove x from the set
// return true if success, false if x overflow
// time complexity: O(1)
func (s *Trends) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

// LoadAndDelete remove x from the set
// loaded report x if in set,ok report x if overflow
// time complexity: O(1)
func (s *Trends) LoadAndDelete(x uint32) (loaded, ok bool) {
	if x > s.getMax() {
		// overflow
		return
	}
	idx, mod := idxMod31(x)
	n := s.getNode()
	if idx >= int(n.getLen()) {
		return false, true
	}
	for {
		n = s.getNode()
		item := n.load(idx)
		if (item>>mod)&1 == 0 {
			return false, true
		}
		if n.evacuted(item) {
			runtime.Gosched()
			continue
		}
		if atomic.CompareAndSwapUint32(&n.data[idx], item, item&^(1<<mod)) {
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
// Range may be O(32*N) with the worst time complexity.
func (s *Trends) Range(f func(x uint32) bool) {
	n := s.getNode()
	sNum := int(n.getLen())
	for i := 0; i < sNum; i++ {
		item := n.load(i)
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

// overflow idx if large len,cap
// if idx=cap,need grow
func (s *Trends) overflow(idx int) bool {
	for {
		n := s.getNode()
		nlen := n.getLen()
		if idx < int(nlen) {
			return false
		}
		// idx > len
		ncap := n.getCap()
		if idx < int(ncap) {
			if atomic.CompareAndSwapUint32(&n.len, nlen, uint32(idx+1)) {
				return false
			}
		}
		// idx > cap, grow work
		if growWork(s, n, uint32(idx+1)) {
			n := s.getNode()
			// check if idx<cap
			return idx >= int(n.getCap())
		}
		runtime.Gosched()
	}
}

func growWork(s *Trends, old *node, cap uint32) bool {
	if !atomic.CompareAndSwapUint32(&old.resize, 0, 1) {
		// other thread growing
		return false
	}
	// caculate new cap
	newCap := caculateCap(old.getCap(), cap)

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

func caculateCap(old, cap uint32) uint32 {
	newCap := old
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
	return newCap
}

func (n *node) evacuted(x uint32) bool {
	return x&freezeBit > 0
}
