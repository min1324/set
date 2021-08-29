package set

import (
	"reflect"
)

const (
	// the max item can store in set.
	// total has slice: 1<<31, each slice can hold 31 item
	// so maximum = 1<<31*31
	// but the memory had not enough space.
	// set the maximum= 1<<24*31
	maximum uint32 = 1 << 24 * 31
)

// Set
type Set interface {
	// Load reports whether the set contains the non-negative value x.
	Load(x uint32) (ok bool)

	// Store adds the non-negative value x to the set.
	// return true if success,or false if x overflow with max
	Store(x uint32) bool

	// Delete remove x from the set
	// return true if success,or false if x overflow with max
	Delete(x uint32) bool

	// LoadOrStore adds the non-negative value x to the set.
	// loaded report x if in set
	// ok if true if success,or false if x overflow with max
	LoadOrStore(x uint32) (loaded, ok bool)

	// LoadAndDelete remove x from the set
	// loaded report x if in set
	// ok if true if success,or false if x overflow with max
	LoadAndDelete(x uint32) (loaded, ok bool)

	// Range calls f sequentially for each item present in the set.
	// If f returns false, range stops the iteration.
	Range(f func(x uint32) bool)
}

// New return a set with items args.
// cap is set cap,if cap<1,will use 256.
func New(cap int, args ...uint32) Set {
	var s IntSet
	s.OnceInit(cap)
	s.Adds(args...)
	return &s
}

// New return a set with items args.
// cap is set cap,if cap<1,will use 256.
func NewSlice(cap int, args ...uint32) Set {
	var s SliceSet
	s.OnceInit(cap)
	s.Adds(args...)
	return &s
}

type reflactType int

const (
	rtOther reflactType = iota
	rtIntSet
	rtSlice
)

var (
	IntType   = reflect.TypeOf(new(IntSet))
	SliceType = reflect.TypeOf(new(SliceSet))
)

// typeEqual check type s and t if equal,
// return true if type s==t,and type of s and t
func typeEqual(s, t Set) (rt reflactType, ok bool) {
	rtS := reflect.TypeOf(s)
	rtT := reflect.TypeOf(t)
	ok = rtS == rtT
	if ok {
		switch rtS {
		case IntType:
			rt = rtIntSet
		case SliceType:
			rt = rtSlice
		default:
			rt = rtOther
		}
	}
	return
}

func items(s Set) []uint32 {
	var a = make([]uint32, 0, 256)
	i := 0
	s.Range(func(x uint32) bool {
		a = append(a, x)
		i += 1
		return true
	})
	if i < len(a) {
		a = a[:i]
	}
	return a
}

// Equal return set if equal, s <==> t
// worst time complexity: O(N)
// best  time complexity: O(1)
func Equal(s, t Set) bool {
	r, ok := typeEqual(s, t)
	if ok {
		if r == rtIntSet {
			ss := s.(*IntSet)
			tt := t.(*IntSet)
			return intSetEqual(ss, tt)
		} else if r == rtSlice {
			ss := s.(*SliceSet)
			tt := t.(*SliceSet)
			return sliceSetEqual(ss, tt)
		}
	}
	es := items(s)
	var i = 0
	var flag = true
	t.Range(func(x uint32) bool {
		if es[i] == x {
			i++
			return true
		}
		flag = false
		return false
	})
	if i < len(es) {
		flag = false
	}
	return flag
}

// Union return the union set of s and t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Union(s, t Set) Set {
	r, ok := typeEqual(s, t)
	if ok {
		if r == rtIntSet {
			ss := s.(*IntSet)
			tt := t.(*IntSet)
			return intSetUnion(ss, tt)
		} else if r == rtSlice {
			ss := s.(*SliceSet)
			tt := t.(*SliceSet)
			return sliceSetUnion(ss, tt)
		}
	}
	var p IntSet
	es := items(s)
	et := items(t)
	maxCap := max(int(es[len(es)-1]), int(et[len(et)-1]))
	p.onceInit(maxCap)

	var i, j = 0, 0
	for j < len(et) && i < len(es) {
		if es[i] == et[j] {
			p.Store(es[i])
			i += 1
			j += 1
		} else if es[i] < et[j] {
			p.Store(es[i])
			i += 1
			continue
		} else {
			p.Store(et[j])
			j += 1
			continue
		}
	}
	for ; i < len(es); i++ {
		p.Store(es[i])
	}
	for ; j < len(et); j++ {
		p.Store(et[j])
	}
	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Intersect(s, t Set) Set {
	r, ok := typeEqual(s, t)
	if ok {
		if r == rtIntSet {
			ss := s.(*IntSet)
			tt := t.(*IntSet)
			return intSetIntersect(ss, tt)
		} else if r == rtSlice {
			ss := s.(*SliceSet)
			tt := t.(*SliceSet)
			return sliceSetIntersect(ss, tt)
		}
	}
	var p IntSet
	es := items(s)
	et := items(t)
	minCap := min(int(es[len(es)-1]), int(et[len(et)-1]))
	p.onceInit(minCap)

	var i, j = 0, 0
	for j < len(et) && i < len(es) {
		if es[i] == et[j] {
			p.Store(es[i])
			i += 1
			j += 1
		} else if es[i] < et[j] {
			i += 1
			continue
		} else {
			j += 1
			continue
		}
	}
	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Difference(s, t Set) Set {
	r, ok := typeEqual(s, t)
	if ok {
		if r == rtIntSet {
			ss := s.(*IntSet)
			tt := t.(*IntSet)
			return intSetDifference(ss, tt)
		} else if r == rtSlice {
			ss := s.(*SliceSet)
			tt := t.(*SliceSet)
			return sliceSetDifference(ss, tt)
		}
	}
	var p IntSet
	es := items(s)
	et := items(t)
	p.onceInit(int(es[len(es)-1]))

	var i, j = 0, 0
	for j < len(et) && i < len(es) {
		if es[i] == et[j] {
			i += 1
			j += 1
		} else if es[i] < et[j] {
			p.Store(es[i])
			i += 1
			continue
		} else {
			j += 1
			continue
		}
	}
	for ; i < len(es); i++ {
		p.Store(es[i])
	}
	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Complement(s, t Set) Set {
	r, ok := typeEqual(s, t)
	if ok {
		if r == rtIntSet {
			ss := s.(*IntSet)
			tt := t.(*IntSet)
			return intSetComplement(ss, tt)
		} else if r == rtSlice {
			ss := s.(*SliceSet)
			tt := t.(*SliceSet)
			return sliceSetComplement(ss, tt)
		}
	}
	var p IntSet
	es := items(s)
	et := items(t)
	maxCap := max(int(es[len(es)-1]), int(et[len(et)-1]))
	p.onceInit(maxCap)

	var i, j = 0, 0
	for j < len(et) && i < len(es) {
		if es[i] == et[j] {
			i += 1
			j += 1
		} else if es[i] < et[j] {
			p.Store(es[i])
			i += 1
			continue
		} else {
			p.Store(et[j])
			j += 1
			continue
		}
	}
	for ; i < len(es); i++ {
		p.Store(es[i])
	}
	for ; j < len(et); j++ {
		p.Store(et[j])
	}
	return &p
}

// Equal return set if equal, s <==> t
// worst time complexity: O(N)
// best  time complexity: O(1)
func intSetEqual(s, t *IntSet) bool {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
	for i := 0; i < minLen; i++ {
		if s.load(i) != t.load(i) {
			return false
		}
	}
	if sLen == tLen {
		return true
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			if s.load(i) != 0 {
				return false
			}
		}
	} else {
		for i := minLen; i < tLen; i++ {
			if t.load(i) != 0 {
				return false
			}
		}
	}
	return true
}

// Union return the union set of s and t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func intSetUnion(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := int(s.getLen()), int(t.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.OnceInit(max(s.Cap(), t.Cap()))

	// [0-minLen]
	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)|t.load(i))
	}
	// [minLen-maxLen]
	if sLen < tLen {
		for i := minLen; i < maxLen; i++ {
			p.store(i, t.load(i))
		}
	} else {
		for i := minLen; i < maxLen; i++ {
			p.store(i, s.load(i))
		}
	}
	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func intSetIntersect(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
	p.OnceInit(min(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&t.load(i))
	}

	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func intSetDifference(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
	p.OnceInit(s.Cap())

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&^t.load(i))
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.store(i, s.load(i))
		}
	}

	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func intSetComplement(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := int(s.getLen()), int(t.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.OnceInit(max(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)^t.load(i))
	}
	for i := minLen; i < maxLen; i++ {
		if sLen > tLen {
			p.store(i, s.load(i))
		} else {
			p.store(i, t.load(i))
		}
	}

	return &p
}

// Equal return set if equal, s <==> t
// worst time complexity: O(N)
// best  time complexity: O(1)
func sliceSetEqual(s, t *SliceSet) bool {
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	minLen := min(sLen, tLen)
	for i := 0; i < minLen; i++ {
		if s.load(i) != t.load(i) {
			return false
		}
	}
	if sLen == tLen {
		return true
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			if s.load(i) != 0 {
				return false
			}
		}
	} else {
		for i := minLen; i < tLen; i++ {
			if t.load(i) != 0 {
				return false
			}
		}
	}
	return true
}

// Union return the union set of s and t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func sliceSetUnion(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.onceInit(max(s.Cap(), t.Cap()))

	// [0-minLen]
	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)|t.load(i))
	}
	// [minLen-maxLen]
	if sLen < tLen {
		for i := minLen; i < maxLen; i++ {
			p.store(i, t.load(i))
		}
	} else {
		for i := minLen; i < maxLen; i++ {
			p.store(i, s.load(i))
		}
	}

	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func sliceSetIntersect(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	minLen := min(sLen, tLen)
	p.onceInit(min(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&t.load(i))
	}
	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func sliceSetDifference(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	minLen := min(sLen, tLen)
	p.onceInit(s.Cap())

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&^t.load(i))
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.store(i, s.load(i))
		}
	}

	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func sliceSetComplement(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.onceInit(max(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)^t.load(i))
	}
	for i := minLen; i < maxLen; i++ {
		if sLen > tLen {
			p.store(i, s.load(i))
		} else {
			p.store(i, t.load(i))
		}
	}

	return &p
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxmin(x, y int) (max, min int) {
	if x > y {
		return x, y
	}
	return y, x
}
