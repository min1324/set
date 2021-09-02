package set_test

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
)

type Interface interface {
	OnceInit(cap int)
	Load(x uint32) bool
	Store(x uint32) bool
	LoadOrStore(x uint32) (loaded, ok bool)
	LoadAndDelete(x uint32) (loaded, ok bool)
	Delete(x uint32) bool
	Range(f func(x uint32) bool)
}

const (
	// platform bit = 2^setBits,(32/64)
	setBits  = 5 //+ (^uint(0) >> 63)
	platform = 1 << setBits
	setMask  = 1<<setBits - 1

	maximum  uint32 = 1 << 24 * 31
	initSize        = 1 << 8
)

type MutexSet struct {
	mu   sync.Mutex
	once sync.Once

	max uint32

	// max input x
	cap uint32

	// len(items)
	len uint32

	items []uint32
}

func (s *MutexSet) onceInit(max int) {
	s.once.Do(func() {
		if max < 1 || max > int(maximum) {
			max = int(maximum)
		}
		var cap uint32 = uint32(max>>5 + 1)
		s.items = make([]uint32, cap)
		s.cap = uint32(cap)
		s.max = uint32(max)
	})
}

// OnceInit initialize set use cap
// it only execute once time.
// if cap<1, will use 256.
func (s *MutexSet) OnceInit(max int) {
	s.onceInit(max)
}

// Init initialize queue use default size: 256
// it only execute once time.
func (s *MutexSet) Init() {
	s.onceInit(initSize)
}

func (s *MutexSet) load(i int) uint32 {
	return atomic.LoadUint32(&s.items[i])
}

func (s *MutexSet) store(i int, x uint32) {
	s.verify(i)
	atomic.StoreUint32(&s.items[i], x)
}

func (q *MutexSet) getLen() uint32 {
	return atomic.LoadUint32(&q.len)
}

func (q *MutexSet) getMax() uint32 {
	return atomic.LoadUint32(&q.max)
}

func (q *MutexSet) getCap() uint32 {
	return atomic.LoadUint32(&q.cap)
}

// Cap return queue's cap
func (q *MutexSet) Cap() int {
	return int(atomic.LoadUint32(&q.max))
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

func (s *MutexSet) Load(x uint32) bool {
	if x > s.getMax() {
		// overflow
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, mod := idxMod(x)
	if idx >= int(s.getLen()) {
		return false
	}
	item := s.load(idx)
	return (item>>mod)&1 == 1

}

func (s *MutexSet) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

func (s *MutexSet) LoadOrStore(x uint32) (loaded, ok bool) {
	s.Init()
	if x > s.getMax() {
		return false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, mod := idxMod(x)
	if !s.verify(idx) {
		return
	}

	item := s.load(idx)
	if (item>>mod)&1 == 1 {
		return true, true
	}
	s.store(idx, item|1<<mod)
	// atomic.StoreUint32(&s.items[idx], item|1<<mod)
	return false, true
}

func (s *MutexSet) verify(idx int) bool {
	slen := int(s.getLen())
	if idx < int(slen) {
		return true
	}
	if idx < int(s.getCap()) {
		atomic.StoreUint32(&s.len, uint32(idx+1))
	} else {
		// grow
		oldCap := atomic.LoadUint32(&s.cap)
		newCap := caculateCap(oldCap, uint32(idx))
		data := make([]uint32, newCap)
		for i := 0; i < int(oldCap); i++ {
			data[i] = s.load(idx)
		}
		s.items = data
		atomic.StoreUint32(&s.len, uint32(idx))
		atomic.StoreUint32(&s.cap, newCap)
	}
	return true
}

func (s *MutexSet) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

func (s *MutexSet) LoadAndDelete(x uint32) (loaded, ok bool) {
	s.Init()
	if x > s.getMax() {
		return false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, mod := idxMod(x)
	if idx >= int(s.getLen()) {
		return false, false
	}
	item := s.load(idx)
	if (item>>mod)&1 == 0 {
		return false, true
	}
	atomic.StoreUint32(&s.items[idx], item&^(1<<mod))
	return true, true
}

func (s *MutexSet) Range(f func(x uint32) bool) {
	slen := s.getLen()
	for i := 0; i < int(slen); i++ {
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

// Size return the number of elements in set
// worst time complexity: worst: O(32*N)
// best  time complexity: O(N)
func (s *MutexSet) Size() int {
	var sum int
	s.Range(func(x uint32) bool {
		sum += 1
		return true
	})
	return sum
}

func (s *MutexSet) Clear() {
	s.mu.Lock()
	for i := 0; i < int(s.getLen()); i++ {
		atomic.StoreUint32(&s.items[i], 0)
	}
	atomic.StoreUint32(&s.len, 0)
	s.mu.Unlock()
}

func (s *MutexSet) Null() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getLen() == 0 {
		return true
	}
	for i := 0; i < int(s.getLen()); i++ {
		if s.load(i) != 0 {
			return false
		}
	}
	return true
}

func (s *MutexSet) Items() []uint32 {
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

func (s *MutexSet) String() string {
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
