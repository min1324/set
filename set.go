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

	// getReflactType return value
	rtIntInt
	rtIntSlice
	rtSliceSlice
	rtSliceInt
)

var (
	IntType   = reflect.TypeOf(new(IntSet))
	SliceType = reflect.TypeOf(new(SliceSet))
)

// get s,t reflact type
func getReflactType(s, t Set) (r reflactType) {
	rtS := reflect.TypeOf(s)
	rtT := reflect.TypeOf(t)
	var rt reflactType
	switch rtS {
	case IntType:
		rt = rtIntSet
	case SliceType:
		rt = rtSlice
	default:
		return rtOther
	}
	switch rtT {
	case IntType:
		if rt == rtIntSet {
			return rtIntInt
		} else {
			return rtSliceInt
		}
	case SliceType:
		if rt == rtIntSet {
			return rtIntSlice
		} else {
			return rtSliceSlice
		}
	default:
		return rtOther
	}
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
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return intSetEqual(ss, tt)
	case rtSliceSlice:
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sliceSetEqual(ss, tt)
	case rtIntSlice:
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return intSetEqual(ss, itt)
	case rtSliceInt:
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return intSetEqual(iss, tt)
	}
	return generalEqual(s, t)
}

// Union return the union set of s and t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Union(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return intSetUnion(ss, tt)
	case rtSliceSlice:
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sliceSetUnion(ss, tt)
	case rtIntSlice:
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return intSetUnion(ss, itt)
	case rtSliceInt:
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return intSetUnion(iss, tt)
	}
	return generalUnion(s, t)
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Intersect(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return intSetIntersect(ss, tt)
	case rtSliceSlice:
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sliceSetIntersect(ss, tt)
	case rtIntSlice:
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return intSetIntersect(ss, itt)
	case rtSliceInt:
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return intSetIntersect(iss, tt)
	}
	return generalIntersect(s, t)
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Difference(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return intSetDifference(ss, tt)
	case rtSliceSlice:
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sliceSetDifference(ss, tt)
	case rtIntSlice:
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return intSetDifference(ss, itt)
	case rtSliceInt:
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return intSetDifference(iss, tt)
	}
	return generalDifference(s, t)

}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Complement(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return intSetComplement(ss, tt)
	case rtSliceSlice:
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sliceSetComplement(ss, tt)
	case rtIntSlice:
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return intSetComplement(ss, itt)
	case rtSliceInt:
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return intSetComplement(iss, tt)
	}
	return generalComplement(s, t)
}

func generalEqual(s, t Set) bool {
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

func generalUnion(s, t Set) Set {
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

func generalIntersect(s, t Set) Set {
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

func generalDifference(s, t Set) Set {
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

func generalComplement(s, t Set) Set {
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

func SliceToInt(s *SliceSet) *IntSet {
	node := s.getNode()
	var n IntSet
	max := s.Cap()
	slen := int(node.getLen())
	n.onceInit(max)
	for i := 0; i < slen; i++ {
		item := node.load(i)
		// u32 实际存位值
		ni := (i + 1) * 31 / 32
		//有效补偿位
		bit := i % 32

		// ni - 补偿
		ivalue := (item &^ (1<<bit - 1)) >> bit
		n.store(ni, n.load(ni)|ivalue)

		// 补偿ni-(i>>5+1)
		if i%32 != 0 {
			bvalue := item & (1<<bit - 1)
			bvalue <<= 31 - ((i - 1) % 32)
			n.store(i-(i>>5+1), n.load(i-(i>>5+1))|bvalue)
		}
	}
	return &n
}

func IntToSlice(s *IntSet) *SliceSet {
	var n SliceSet
	slen := int(s.getLen())
	nCap := (slen + slen/31) * 32
	n.onceInit(nCap)
	n.max = uint32(maximum)

	for i := 0; i < slen; i++ {
		item := s.load(i)
		// u32 实际存位值
		ni := i + i/31
		//有效补偿位
		bit := i % 31
		ivalue := item << bit
		// 去掉bit最高位
		ivalue &^= 1 << 31
		n.store(ni, n.load(ni)|ivalue)

		// 补偿i+(i/31+1)
		bv := item >> (31 - bit)
		n.store(ni+1, n.load(ni+1)|bv)
	}
	return &n
}
