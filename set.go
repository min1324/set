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
	// OnceInit once time with max item.
	OnceInit(max int)

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
	rtOtherSame
)

var (
	// reflact.typeof IntSet
	IntType = reflect.TypeOf(new(IntSet))
	// reflact.typeof SliceSet
	SliceType = reflect.TypeOf(new(SliceSet))
)

// get s,t reflact type
func getReflactType(s, t Set) (r reflactType) {
	rtS := reflect.TypeOf(s)
	rtT := reflect.TypeOf(t)
	var rt reflactType

	ok := rtS == rtT
	if ok {
		switch rtS {
		case IntType:
			return rtIntInt
		case SliceType:
			return rtSliceSlice
		default:
			return rtOtherSame
		}
	} else {
		// s,t not same type
		switch rtS {
		case IntType:
			rt = rtIntSet
		case SliceType:
			rt = rtSlice
		default:
			if rtT == rtS {
				return rtOtherSame
			}
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
func Equal(s, t Set) bool {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return sameTypeEqual(ss, tt)
	case rtSliceSlice:
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sameTypeEqual(ss, tt)
	case rtIntSlice:
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return sameTypeEqual(ss, itt)
	case rtSliceInt:
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return sameTypeEqual(iss, tt)
	case rtOtherSame:
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
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return sameTypeUnion(ss, tt, &p)
	case rtSliceSlice:
		var p SliceSet
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sameTypeUnion(ss, tt, &p)
	case rtIntSlice:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return sameTypeUnion(ss, itt, &p)
	case rtSliceInt:
		var p IntSet
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return sameTypeUnion(iss, tt, &p)
	case rtOtherSame:
		typ := reflect.TypeOf(s)
		p := reflect.New(typ.Elem()).Interface().(Set)
		return generalUnion(s, t, p)
	}
	var p IntSet
	return generalUnion(s, t, &p)
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Intersect(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return sameTypeIntersect(ss, tt, &p)
	case rtSliceSlice:
		var p SliceSet
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sameTypeIntersect(ss, tt, &p)
	case rtIntSlice:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return sameTypeIntersect(ss, itt, &p)
	case rtSliceInt:
		var p IntSet
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return sameTypeIntersect(iss, tt, &p)
	case rtOtherSame:
		typ := reflect.TypeOf(s)
		p := reflect.New(typ.Elem()).Interface().(Set)
		return generalIntersect(s, t, p)
	}
	var p IntSet
	return generalIntersect(s, t, &p)
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Difference(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return sameTypeDifference(ss, tt, &p)
	case rtSliceSlice:
		var p SliceSet
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sameTypeDifference(ss, tt, &p)
	case rtIntSlice:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return sameTypeDifference(ss, itt, &p)
	case rtSliceInt:
		var p IntSet
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return sameTypeDifference(iss, tt, &p)
	case rtOtherSame:
		typ := reflect.TypeOf(s)
		p := reflect.New(typ.Elem()).Interface().(Set)
		return generalDifference(s, t, p)
	}
	var p IntSet
	return generalDifference(s, t, &p)

}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Complement(s, t Set) Set {
	r := getReflactType(s, t)
	switch r {
	case rtIntInt:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*IntSet)
		return sameTypeComplement(ss, tt, &p)
	case rtSliceSlice:
		var p SliceSet
		ss := s.(*SliceSet)
		tt := t.(*SliceSet)
		return sameTypeComplement(ss, tt, &p)
	case rtIntSlice:
		var p IntSet
		ss := s.(*IntSet)
		tt := t.(*SliceSet)
		itt := SliceToInt(tt)
		return sameTypeComplement(ss, itt, &p)
	case rtSliceInt:
		var p IntSet
		ss := s.(*SliceSet)
		tt := t.(*IntSet)
		iss := SliceToInt(ss)
		return sameTypeComplement(iss, tt, &p)
	case rtOtherSame:
		typ := reflect.TypeOf(s)
		p := reflect.New(typ.Elem()).Interface().(Set)
		return generalComplement(s, t, p)
	}
	var p IntSet
	return generalComplement(s, t, &p)
}

func generalEqual(s, t Set) bool {
	es := items(s)
	et := items(t)
	slen, tlen := len(es), len(et)
	if slen != tlen {
		return false
	}
	for i := 0; i < slen; i++ {
		if es[i] != et[i] {
			return false
		}
	}
	return true
}

func generalUnion(s, t, p Set) Set {
	es := items(s)
	et := items(t)
	maxCap := max(int(es[len(es)-1]), int(et[len(et)-1]))
	p.OnceInit(maxCap)

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
	return p
}

func generalIntersect(s, t, p Set) Set {
	es := items(s)
	et := items(t)
	minCap := min(int(es[len(es)-1]), int(et[len(et)-1]))
	p.OnceInit(minCap)

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
	return p
}

func generalDifference(s, t, p Set) Set {
	es := items(s)
	et := items(t)
	p.OnceInit(int(es[len(es)-1]))

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
	return p
}

func generalComplement(s, t, p Set) Set {
	es := items(s)
	et := items(t)
	maxCap := max(int(es[len(es)-1]), int(et[len(et)-1]))
	p.OnceInit(maxCap)

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
	return p
}

// use for sameType op
type opSet interface {
	Set
	getLen() uint32
	getMax() uint32
	load(int) uint32
	store(int, uint32)
	onceInit(max int)
}

// x,y,p must the same type
// return p

// Equal return set if equal, s <==> t
func sameTypeEqual(s, t opSet) bool {
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
func sameTypeUnion(s, t, p opSet) Set {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.onceInit(max(int(s.getMax()), int(t.getMax())))

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
	return p
}

// Intersect return the intersection set of s and t
func sameTypeIntersect(s, t, p opSet) Set {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
	p.onceInit(min(int(s.getMax()), int(t.getMax())))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&t.load(i))
	}
	return p
}

// Difference return the difference set of s and t
func sameTypeDifference(s, t, p opSet) Set {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
	p.onceInit(int(s.getMax()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&^t.load(i))
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.store(i, s.load(i))
		}
	}

	return p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
func sameTypeComplement(s, t, p opSet) Set {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.onceInit(max(int(s.getMax()), int(t.getMax())))

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

	return p
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

// ToInt convert Set to int set
func ToInt(s Set) *IntSet {
	rt := reflect.TypeOf(s)
	switch rt {
	case IntType:
		// TODO copy s?
		return s.(*IntSet)
	case SliceType:
		ss := s.(*SliceSet)
		return SliceToInt(ss)
	default:
		array := items(s)
		slen := len(array)
		scap := array[slen-1]
		t := New(int(scap))
		for i := 0; i < slen; i++ {
			t.Store(array[i])
		}
		return t.(*IntSet)
	}
}

// ToSlice convert Set to slice set
func ToSlice(s Set) *SliceSet {
	rt := reflect.TypeOf(s)
	switch rt {
	case IntType:
		ss := s.(*IntSet)
		return IntToSlice(ss)
	case SliceType:
		// TODO copy s?
		return s.(*SliceSet)
	default:
		array := items(s)
		slen := len(array)
		scap := array[slen-1]
		t := NewSlice(int(scap))
		for i := 0; i < slen; i++ {
			t.Store(array[i])
		}
		return t.(*SliceSet)
	}
}

// SliceToInt convert slice set to int set
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

// SliceToInt convert int set to slice set
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

// // Equal return set if equal, s <==> t
// func intSetEqual(s, t *IntSet) bool {
// 	sLen, tLen := int(s.getLen()), int(t.getLen())
// 	minLen := min(sLen, tLen)
// 	for i := 0; i < minLen; i++ {
// 		if s.load(i) != t.load(i) {
// 			return false
// 		}
// 	}
// 	if sLen == tLen {
// 		return true
// 	}
// 	if sLen > tLen {
// 		for i := minLen; i < sLen; i++ {
// 			if s.load(i) != 0 {
// 				return false
// 			}
// 		}
// 	} else {
// 		for i := minLen; i < tLen; i++ {
// 			if t.load(i) != 0 {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// // Union return the union set of s and t.
// func intSetUnion(s, t *IntSet) *IntSet {
// 	var p IntSet
// 	sLen, tLen := int(s.getLen()), int(t.getLen())
// 	maxLen, minLen := maxmin(sLen, tLen)
// 	p.OnceInit(max(s.Cap(), t.Cap()))

// 	// [0-minLen]
// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)|t.load(i))
// 	}
// 	// [minLen-maxLen]
// 	if sLen < tLen {
// 		for i := minLen; i < maxLen; i++ {
// 			p.store(i, t.load(i))
// 		}
// 	} else {
// 		for i := minLen; i < maxLen; i++ {
// 			p.store(i, s.load(i))
// 		}
// 	}
// 	return &p
// }

// // Intersect return the intersection set of s and t
// func intSetIntersect(s, t *IntSet) *IntSet {
// 	var p IntSet
// 	sLen, tLen := int(s.getLen()), int(t.getLen())
// 	minLen := min(sLen, tLen)
// 	p.OnceInit(min(s.Cap(), t.Cap()))

// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)&t.load(i))
// 	}

// 	return &p
// }

// // Difference return the difference set of s and t
// func intSetDifference(s, t *IntSet) *IntSet {
// 	var p IntSet
// 	sLen, tLen := int(s.getLen()), int(t.getLen())
// 	minLen := min(sLen, tLen)
// 	p.OnceInit(s.Cap())

// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)&^t.load(i))
// 	}
// 	if sLen > tLen {
// 		for i := minLen; i < sLen; i++ {
// 			p.store(i, s.load(i))
// 		}
// 	}

// 	return &p
// }

// // Complement return the complement set of s and t
// // item in s but not in t, and not in s but in t.
// func intSetComplement(s, t *IntSet) *IntSet {
// 	var p IntSet
// 	sLen, tLen := int(s.getLen()), int(t.getLen())
// 	maxLen, minLen := maxmin(sLen, tLen)
// 	p.OnceInit(max(s.Cap(), t.Cap()))

// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)^t.load(i))
// 	}
// 	for i := minLen; i < maxLen; i++ {
// 		if sLen > tLen {
// 			p.store(i, s.load(i))
// 		} else {
// 			p.store(i, t.load(i))
// 		}
// 	}

// 	return &p
// }

// // Equal return set if equal, s <==> t
// func sliceSetEqual(s, t *SliceSet) bool {
// 	sn, tn := s.getNode(), t.getNode()
// 	sLen, tLen := int(sn.getLen()), int(tn.getLen())
// 	minLen := min(sLen, tLen)
// 	for i := 0; i < minLen; i++ {
// 		if s.load(i) != t.load(i) {
// 			return false
// 		}
// 	}
// 	if sLen == tLen {
// 		return true
// 	}
// 	if sLen > tLen {
// 		for i := minLen; i < sLen; i++ {
// 			if s.load(i) != 0 {
// 				return false
// 			}
// 		}
// 	} else {
// 		for i := minLen; i < tLen; i++ {
// 			if t.load(i) != 0 {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// // Union return the union set of s and t.
// func sliceSetUnion(s, t *SliceSet) *SliceSet {
// 	var p SliceSet
// 	sn, tn := s.getNode(), t.getNode()
// 	sLen, tLen := int(sn.getLen()), int(tn.getLen())
// 	maxLen, minLen := maxmin(sLen, tLen)
// 	p.onceInit(max(s.Cap(), t.Cap()))

// 	// [0-minLen]
// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)|t.load(i))
// 	}
// 	// [minLen-maxLen]
// 	if sLen < tLen {
// 		for i := minLen; i < maxLen; i++ {
// 			p.store(i, t.load(i))
// 		}
// 	} else {
// 		for i := minLen; i < maxLen; i++ {
// 			p.store(i, s.load(i))
// 		}
// 	}

// 	return &p
// }

// // Intersect return the intersection set of s and t
// func sliceSetIntersect(s, t *SliceSet) *SliceSet {
// 	var p SliceSet
// 	sn, tn := s.getNode(), t.getNode()
// 	sLen, tLen := int(sn.getLen()), int(tn.getLen())
// 	minLen := min(sLen, tLen)
// 	p.onceInit(min(s.Cap(), t.Cap()))

// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)&t.load(i))
// 	}
// 	return &p
// }

// // Difference return the difference set of s and t
// func sliceSetDifference(s, t *SliceSet) *SliceSet {
// 	var p SliceSet
// 	sn, tn := s.getNode(), t.getNode()
// 	sLen, tLen := int(sn.getLen()), int(tn.getLen())
// 	minLen := min(sLen, tLen)
// 	p.onceInit(s.Cap())

// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)&^t.load(i))
// 	}
// 	if sLen > tLen {
// 		for i := minLen; i < sLen; i++ {
// 			p.store(i, s.load(i))
// 		}
// 	}

// 	return &p
// }

// // Complement return the complement set of s and t
// // item in s but not in t, and not in s but in t.
// func sliceSetComplement(s, t *SliceSet) *SliceSet {
// 	var p SliceSet
// 	sn, tn := s.getNode(), t.getNode()
// 	sLen, tLen := int(sn.getLen()), int(tn.getLen())
// 	maxLen, minLen := maxmin(sLen, tLen)
// 	p.onceInit(max(s.Cap(), t.Cap()))

// 	for i := 0; i < minLen; i++ {
// 		p.store(i, s.load(i)^t.load(i))
// 	}
// 	for i := minLen; i < maxLen; i++ {
// 		if sLen > tLen {
// 			p.store(i, s.load(i))
// 		} else {
// 			p.store(i, t.load(i))
// 		}
// 	}

// 	return &p
// }
