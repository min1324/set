package set_test

import (
	"sync"
)

type setInterface interface {
	Len() int
	Load(x uint32) bool
	Store(x uint32)
	LoadOrStore(x uint32) (loaded bool)
	LoadAndDelete(x uint32) (loaded bool)
	Delete(x uint32)
	Range(f func(x uint32) bool)
}

const (
	// platform bit = 2^setBits,(32/64)
	setBits  = 5 //+ (^uint(0) >> 63)
	platform = 1 << setBits
	setMesk  = 1<<setBits - 1
)

type MutexSet struct {
	mu    sync.Mutex
	items []uint32
}

func idxMod(x uint32) (idx, mod int) {
	return int(x >> setBits), int(x & setMesk)
}

func (s *MutexSet) Load(x uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, mod := idxMod(x)
	if idx >= len(s.items) {
		return false
	}
	return (s.items[idx]>>mod)&1 == 1

}

func (s *MutexSet) Store(x uint32) {
	s.LoadOrStore(x)
}

func (s *MutexSet) LoadOrStore(x uint32) (loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, mod := idxMod(x)
	if idx >= len(s.items) {
		for {
			if idx < len(s.items) {
				break
			}
			s.items = append(s.items, 0)
		}
	}
	item := s.items[idx]
	if (item>>mod)&1 == 1 {
		return true
	}
	s.items[idx] |= 1 << mod
	return
}

func (s *MutexSet) Delete(x uint32) {
	s.LoadAndDelete(x)
}

func (s *MutexSet) LoadAndDelete(x uint32) (loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, mod := idxMod(x)
	if idx >= len(s.items) {
		return false
	}
	item := s.items[idx]
	if (item>>mod)&1 == 0 {
		return false
	}
	s.items[idx] &^= 1 << mod
	return true
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
