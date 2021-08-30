package set_test

import (
	"sync"
	"sync/atomic"
)

type Interface interface {
	OnceInit(cap int)
	Cap() int
	Len() int
	Clear()
	Null() bool
	Load(x uint32) bool
	Store(x uint32) bool
	LoadOrStore(x uint32) (loaded, ok bool)
	LoadAndDelete(x uint32) (loaded, ok bool)
	Delete(x uint32) bool
	Range(f func(x uint32) bool)
	Items() []uint32
}

const (
	// platform bit = 2^setBits,(32/64)
	setBits  = 5 //+ (^uint(0) >> 63)
	platform = 1 << setBits
	setMesk  = 1<<setBits - 1

	maxItem  uint32 = 1 << 24 * 31
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
		if max < 1 || max > int(maxItem) {
			max = int(maxItem)
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
func (s *MutexSet) OnceInit(cap int) {
	s.onceInit(cap)
}

// Init initialize queue use default size: 256
// it only execute once time.
func (s *MutexSet) Init() {
	s.onceInit(0)
}

func (s *MutexSet) load(i int) uint32 {
	return atomic.LoadUint32(&s.items[i])
}

func (q *MutexSet) getLen() uint32 {
	return atomic.LoadUint32(&q.len)
}

func (q *MutexSet) getCap() uint32 {
	return atomic.LoadUint32(&q.cap)
}

// Cap return queue's cap
func (q *MutexSet) Cap() int {
	return int(atomic.LoadUint32(&q.max))
}

// Cap return queue's cap
func (q *MutexSet) Max() uint32 {
	return atomic.LoadUint32(&q.max)
}

func idxMod(x uint32) (idx, mod int) {
	return int(x >> setBits), int(x & setMesk)
}

func (s *MutexSet) Load(x uint32) bool {
	if x > s.Max() {
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
	if x > s.Max() {
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
	atomic.StoreUint32(&s.items[idx], item|1<<mod)
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
		newCap := oldCap
		doubleCap := newCap << 1
		if uint32(idx) > doubleCap {
			newCap = uint32(idx)
		} else {
			if newCap < 1024 {
				newCap = doubleCap
			} else {
				// Check 0 < newcap to detect overflow
				// and prevent an infinite loop.
				for 0 < newCap && newCap < uint32(idx) {
					newCap += newCap / 4
				}
				// Set newcap to the requested cap when
				// the newcap calculation overflowed.
				if newCap <= 0 {
					newCap = uint32(idx)
				}
			}
		}
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
	if x > s.Max() {
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

// Len return the number of elements in set
// worst time complexity: worst: O(32*N)
// best  time complexity: O(N)
func (s *MutexSet) Len() int {
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
