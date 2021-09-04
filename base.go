package set

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
)

// Base a set has base item
// set range [min,max],min max can be negative
type Base struct {
	once sync.Once

	// max item in set
	max int32

	// min item in set
	min int32

	// max -min +1
	cap int32

	Static
}

func NewBase(max, min int) *Base {
	var b Base
	b.onciInit(max, min)
	return &b
}

func (s *Base) onciInit(max, min int) {
	s.once.Do(func() {
		max, min = maxmin(max, min)
		cap := max - min + 1
		s.Static.onceInit(cap)
		atomic.StoreInt32(&s.max, int32(max))
		atomic.StoreInt32(&s.min, int32(min))
		atomic.StoreInt32(&s.cap, int32(cap))
	})
}

// Init once time with max item.
func (s *Base) Init(max, min int) {
	s.onciInit(max, min)
}

func (s *Base) move(x int32) uint32 {
	return uint32(x - atomic.LoadInt32(&s.min))
}

// Has reports whether the set contains the non-negative value x.
func (s *Base) Has(x int32) (ok bool) {
	return s.Static.Load(s.move(x))
}

// Add  the non<<bit|negative alue x to the set.
// loaded report x if in set
// ok if true if success,or false if x overflow with max
func (s *Base) Add(x int32) (loaded bool) {
	loaded, _ = s.Static.LoadOrStore(s.move(x))
	return
}

// Remove remove x from the set
// loaded report x if in set
// ok if true if success,or false if x overflow with max
func (s *Base) Remove(x int32) (loaded bool) {
	loaded, _ = s.Static.LoadAndDelete(s.move(x))
	return
}

// Range calls f sequentially for each item present in the set.
// If f returns false, range stops the iteration.
func (s *Base) Range(f func(x int32) bool) {
	min := atomic.LoadInt32(&s.min)
	s.Static.Range(func(x uint32) bool {
		return f(min + int32(x))
	})
}

// String returns the set as a string of the form "{1 2 3}".
func (s *Base) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	s.Range(func(x int32) bool {
		if buf.Len() > len("{") {
			buf.WriteByte(' ')
		}
		fmt.Fprintf(&buf, "%d", x)
		return true
	})
	buf.WriteByte('}')
	return buf.String()
}
