package set_test

import (
	"sync"
	"sync/atomic"
)

type setInterface interface {
	Len() int
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
	setMesk  = 1<<setBits - 1
)

type MutexSet struct {
	mu   sync.Mutex
	once sync.Once

	// max input x
	cap uint32

	// len(items)
	num uint32

	items []uint32
}

const (
	initSize = 1 << 8
)

func (s *MutexSet) onceInit(cap int) {
	s.once.Do(func() {
		if cap < 1 {
			cap = initSize
		}
		num := cap>>5 + 1
		s.items = make([]uint32, num)
		s.num = uint32(num)
		s.cap = uint32(cap)
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

// Cap return queue's cap
func (q *MutexSet) Cap() int {
	return int(atomic.LoadUint32(&q.cap))
}

func idxMod(x uint32) (idx, mod int) {
	return int(x >> setBits), int(x & setMesk)
}

func (s *MutexSet) maxIndex() int {
	return int(atomic.LoadUint32(&s.num))
}

func (s *MutexSet) Load(x uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, mod := idxMod(x)
	if idx >= s.maxIndex() {
		return false
	}
	return (s.items[idx]>>mod)&1 == 1

}

func (s *MutexSet) Store(x uint32) bool {
	_, ok := s.LoadOrStore(x)
	return ok
}

func (s *MutexSet) LoadOrStore(x uint32) (loaded, ok bool) {
	s.Init()
	if x > atomic.LoadUint32(&s.cap) {
		return false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if x > atomic.LoadUint32(&s.cap) {
		return false, false
	}

	idx, mod := idxMod(x)
	if idx >= s.maxIndex() {
		return false, false
	}

	item := s.items[idx]
	if (item>>mod)&1 == 1 {
		return true, true
	}
	s.items[idx] |= 1 << mod
	return false, true
}

func (s *MutexSet) Delete(x uint32) bool {
	_, ok := s.LoadAndDelete(x)
	return ok
}

func (s *MutexSet) LoadAndDelete(x uint32) (loaded, ok bool) {
	s.Init()
	if x > atomic.LoadUint32(&s.cap) {
		return false, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if x > atomic.LoadUint32(&s.cap) {
		return false, false
	}

	idx, mod := idxMod(x)
	if idx >= s.maxIndex() {
		return false, false
	}
	item := s.items[idx]
	if (item>>mod)&1 == 0 {
		return false, true
	}
	s.items[idx] &^= 1 << mod
	return true, true
}

func (s *MutexSet) Range(f func(x uint32) bool) {
	slen := len(s.items)
	for i := 0; i < slen; i++ {
		item := s.items[i]
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
