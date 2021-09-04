package set

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
)

type Map struct {
	count uint32
	sync.Map
}

type setNil *struct{}

// Has reports whether the set contains x
func (s *Map) Has(x interface{}) (ok bool) {
	_, ok = s.Map.Load(x)
	return ok
}

// Add add a value x into set
func (s *Map) Add(x interface{}) (loaded bool) {
	_, ok := s.Map.LoadOrStore(x, setNil(nil))
	if ok {
		atomic.AddUint32(&s.count, 1)
	}
	return ok
}

// Remove remove x from the set
func (s *Map) Remove(x interface{}) (loaded bool) {
	_, ok := s.Map.LoadAndDelete(x)
	if ok {
		atomic.AddUint32(&s.count, ^uint32(0))
	}
	return ok
}

// Range calls f sequentially for each item present in the set.
// If f returns false, range stops the iteration.
func (s *Map) Range(f func(x interface{}) bool) {
	s.Map.Range(func(key, value interface{}) bool {
		return f(key)
	})
}

// String returns the set as a string of the form "{1 2 3}".
func (s *Map) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	s.Range(func(x interface{}) bool {
		if buf.Len() > len("{") {
			buf.WriteByte(' ')
		}
		fmt.Fprintf(&buf, "%d", x)
		return true
	})
	buf.WriteByte('}')
	return buf.String()
}
