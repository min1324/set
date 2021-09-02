package set

// common public function

import (
	"bytes"
	"fmt"
	"reflect"
	"sync/atomic"
	"unsafe"
)

var (
	// reflact.typeof Static
	StaticType = reflect.TypeOf(new(Static))
	// reflact.typeof Trends
	TrendsType = reflect.TypeOf(new(Dynamic))
	// reflact.typeof Option
	// OptionType = reflect.TypeOf(new(Option))
)

// String returns the set as a string of the form "{1 2 3}".
func String(s Set) string {
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

// String returns the set as a string of the form "{1 2 3}".
func (s *Static) String() string { return String(s) }

// String returns the set as a string of the form "{1 2 3}".
func (s *Dynamic) String() string { return String(s) }

// // String returns the set as a string of the form "{1 2 3}".
// func (s *Option) String() string { return String(s) }

// New return a set with items args.
// the frist arg is cap,
// if no arg or cap<1,return an init Trends set.
// or cap>1,return an init Static set.
func New(args ...uint32) Set {
	if len(args) > 0 {
		cap := args[0]
		if 1 < cap {
			return NewStatic(int(cap), args[1:]...)
		}
		return NewDynamic(int(cap), args[1:]...)
	}
	return NewDynamic(0)
}

// NewStatic return a set with items args.
// cap is set cap,if cap<1,will use 256.
func NewStatic(cap int, args ...uint32) Set {
	var s Static
	s.OnceInit(cap)
	Adds(&s, args...)
	return &s
}

// NewDynamic return a set with items args.
// cap is set cap,if cap<1,will use 256.
func NewDynamic(cap int, args ...uint32) Set {
	var s Dynamic
	s.OnceInit(cap)
	Adds(&s, args...)
	return &s
}

type reflactType int

const (
	// reflactType move bit
	bit = 16
)

const (
	rtStatic reflactType = iota + 1
	rtDynamic
	// rtOption
	rtOther

	// two type relation.
	rtStaticStatic  = rtStatic<<bit | rtStatic
	rtStaticDynamic = rtStatic<<bit | rtDynamic
	// rtStaticOption = rtStatic<<bit | rtOption

	rtDynamicStatic  = rtDynamic<<bit | rtStatic
	rtDynamicDynamic = rtDynamic<<bit | rtDynamic
	// rtTrendsOption = rtTrends<<bit | rtOption

	// rtOptionStatic = rtOption<<bit | rtStatic
	// rtOptionTrends = rtOption<<bit | rtTrends
	// rtOptionOption = rtOption<<bit | rtOption

	rtOtherSame = rtOther<<bit | rtOther

	// other case is two type not the same or know.
	// exp: rtOtherNoSame = rtOther<<bit | rtOption
)

// reflactType = s<<bit | t,keep sync with const
func add(s, t reflactType) reflactType {
	return s<<bit | t
}

// opGeneral generalOperation func type
type opGeneral func(s, t, p Set) Set

// opSameType sameTypeOperation func type
type opSameType func(s, t, p opSet) opSet

// cbFunc call back func in readOnlyCB
type cbFunc func(x, y Set, flag opFlag, sameType opSameType) Set

type opFlag int

const (
	opUnion opFlag = iota
	opIntersect
	opDifference
	opComplement
)

func opGetMax(a, b uint32, flag opFlag) int {
	switch flag {
	case opUnion:
		return max(int(a), int(b))
	case opIntersect:
		return min(int(a), int(b))
	case opDifference:
		return int(a)
	case opComplement:
		return max(int(a), int(b))
	}
	return int(maximum)
}

// readOnlyCB return cb func depend on two type relation.
// if not in map,two type is otherSame or otherNoSame
var readOnlyCB map[reflactType]cbFunc = map[reflactType]cbFunc{
	rtStaticStatic: func(x, y Set, flag opFlag, sameType opSameType) Set {
		xx := x.(*Static)
		yy := y.(*Static)
		var p Static
		cap := opGetMax(xx.getMax(), yy.getMax(), flag)
		p.OnceInit(cap)
		sameType(xx, yy, &p)
		return &p
	},
	rtStaticDynamic: func(x, y Set, flag opFlag, sameType opSameType) Set {
		xx := x.(*Static)
		yy := y.(*Dynamic)
		cy := trendsToStatic(yy)
		var p Static
		cap := opGetMax(xx.getMax(), yy.getMax(), flag)
		p.OnceInit(cap)
		sameType(xx, cy, &p)
		return &p
	},
	// rtStaticOption: func(x, y Set, flag opFlag, sameType opSameType) Set {
	// 	xx := x.(*Static)
	// 	yy := y.(*Option)
	// 	cy := yy.Static()
	// 	var p Static
	// 	cap := opGetMax(xx.getMax(), yy.getMax(), flag)
	// 	p.OnceInit(cap)
	// 	sameType(xx, cy, &p)
	// 	return &p
	// },
	rtDynamicDynamic: func(x, y Set, flag opFlag, sameType opSameType) Set {
		xx := x.(*Dynamic)
		yy := y.(*Dynamic)
		var p Dynamic
		cap := opGetMax(xx.getMax(), yy.getMax(), flag)
		p.OnceInit(cap)
		sameType(xx, yy, &p)
		return &p
	},
	rtDynamicStatic: func(x, y Set, flag opFlag, sameType opSameType) Set {
		xx := x.(*Dynamic)
		yy := y.(*Static)
		cx := trendsToStatic(xx)
		var p Static
		cap := opGetMax(xx.getMax(), yy.getMax(), flag)
		p.OnceInit(cap)
		sameType(cx, yy, &p)
		return &p
	},
	// rtTrendsOption: func(x, y Set, flag opFlag, sameType opSameType) Set {
	// 	xx := x.(*Trends)
	// 	yy := y.(*Option)
	// 	cy := yy.Trends()
	// 	var p Trends
	// 	cap := opGetMax(xx.getMax(), yy.getMax(), flag)
	// 	p.OnceInit(cap)
	// 	sameType(xx, cy, &p)
	// 	return &p
	// },
	// rtOptionStatic: func(x, y Set, flag opFlag, sameType opSameType) Set {
	// 	xx := x.(*Option)
	// 	yy := y.(*Static)
	// 	cx := xx.Static()
	// 	var p Static
	// 	cap := opGetMax(xx.getMax(), yy.getMax(), flag)
	// 	p.OnceInit(cap)
	// 	sameType(cx, yy, &p)
	// 	return &p
	// },
	// rtOptionTrends: func(x, y Set, flag opFlag, sameType opSameType) Set {
	// 	xx := x.(*Option)
	// 	yy := y.(*Trends)
	// 	cx := xx.Trends()
	// 	var p Trends
	// 	cap := opGetMax(xx.getMax(), yy.getMax(), flag)
	// 	p.OnceInit(cap)
	// 	sameType(cx, yy, &p)
	// 	return &p
	// },
	// rtOptionOption: func(x, y Set, flag opFlag, sameType opSameType) Set {
	// 	xx := x.(*Option)
	// 	yy := y.(*Option)
	// 	xe := xx.getEntry()
	// 	ye := yy.getEntry()
	// 	cap := opGetMax(xx.getMax(), yy.getMax(), flag)

	// 	if xe.typ == ye.typ || (xe.typ == optEntry15 && ye.typ == optEntry16) ||
	// 		xe.typ == optEntry16 && ye.typ == optEntry15 {
	// 		var p Option
	// 		p.init(cap, xe.typ)
	// 		sameType(xx, yy, &p)
	// 		return &p
	// 	}

	// 	// x,y 不同,31+32,15+31,15+32,bit+31,bit+32
	// 	// 转成32
	// 	var p Option
	// 	p.init(cap, optEntry32)

	// 	m, n := xx.Static(), yy.Static()
	// 	sameType(m, n, &p)
	// 	return &p
	// },
}

// get s,t reflact type, return two type relation.
func getReflectType(s, t Set) (r reflactType) {
	rts := reflect.TypeOf(s)
	rtt := reflect.TypeOf(t)
	var ss reflactType
	switch rts {
	case StaticType:
		ss = rtStatic
	case TrendsType:
		ss = rtDynamic
	// case OptionType:
	// 	ss = rtOption
	default:
		ss = rtOther
	}
	var tt reflactType
	switch rtt {
	case StaticType:
		tt = rtStatic
	case TrendsType:
		tt = rtDynamic
	// case OptionType:
	// 	tt = rtOption
	default:
		tt = rtOther
	}
	return add(ss, tt)
}

// operation two set operate with union,intersect,diffrence,complement
func operation(x, y Set, flag opFlag, sameType opSameType, general opGeneral) Set {
	r := getReflectType(x, y)

	// set x,y is know type,use same type methor
	if f, ok := readOnlyCB[r]; ok {
		return f(x, y, flag, sameType)
	}
	if r == rtOtherSame {
		typ := reflect.TypeOf(x)
		p := reflect.New(typ.Elem()).Interface().(Set)
		return general(x, y, p)
	}
	// use general methor
	var p Static
	return general(x, y, &p)
}

// Union return the union set of s and t.
// if set s and t is same type,return the same type
// if not the same,return the Static type
//
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Union(s, t Set) Set {
	return operation(s, t, opUnion, sameTypeUnion, generalUnion)
}

// Intersect return the intersection set of s and t
// item in s and t
// if set s and t is same type,return the same type
// if not the same,return the Static type
//
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Intersect(s, t Set) Set {
	return operation(s, t, opIntersect, sameTypeIntersect, generalIntersect)
}

// Difference return the difference set of s and t
// item in s and not in t
// if set s and t is same type,return the same type
// if not the same,return the Static type
//
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Difference(s, t Set) Set {
	return operation(s, t, opDifference, sameTypeDifference, generalDifference)
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// if set s and t is same type,return the same type
// if not the same,return the Static type
//
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Complement(s, t Set) Set {
	return operation(s, t, opComplement, sameTypeComplement, generalComplement)
}

// Equal return set if equal, s <==> t
func Equal(s, t Set) bool {
	r := getReflectType(s, t)
	switch r {
	case rtStaticStatic:
		ss := s.(*Static)
		tt := t.(*Static)
		return sameTypeEqual(ss, tt)
	case rtDynamicDynamic:
		ss := s.(*Dynamic)
		tt := t.(*Dynamic)
		return sameTypeEqual(ss, tt)
	case rtStaticDynamic:
		ss := s.(*Static)
		tt := t.(*Dynamic)
		itt := trendsToStatic(tt)
		return sameTypeEqual(ss, itt)
	case rtDynamicStatic:
		ss := s.(*Dynamic)
		tt := t.(*Static)
		iss := trendsToStatic(ss)
		return sameTypeEqual(iss, tt)
	case rtOtherSame:
	}
	return generalEqual(s, t)
}

// Copy return a copy of s
//
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Copy(s Set) Set {
	r := reflect.TypeOf(s)
	switch r {
	case StaticType:
		ss := s.(*Static)
		var p Static
		p.OnceInit(int(ss.getMax()))
		sameTypeCopy(ss, &p)
		return &p
	case TrendsType:
		ss := s.(*Dynamic)
		var p Dynamic
		p.OnceInit(int(ss.getMax()))
		sameTypeCopy(ss, &p)
		return &p
		// case OptionType:
		// 	ss := s.(*Option)
		// 	n := ss.getEntry()
		// 	var p Option
		// 	p.init(int(ss.getMax()), n.typ)
		// 	sameTypeCopy(ss, &p)
		// 	return &p
	}
	typ := reflect.TypeOf(s)
	p := reflect.New(typ.Elem()).Interface().(Set)
	return generalCopy(s, p)
}

// Items return all element in the set
// time complexity: O(N)
func Items(s Set) []uint32 {
	sum := 0
	var slen uint32 = 0
	r := reflect.TypeOf(s)
	switch r {
	case StaticType:
		ss := s.(*Static)
		slen = ss.getLen()
	case TrendsType:
		ss := s.(*Dynamic)
		slen = ss.getEntry().getLen()
	// case OptionType:
	// 	ss := s.(*Option)
	// 	slen = ss.getEntry().getLen()
	default:
		slen = initSize
	}
	array := make([]uint32, 0, slen*32)
	s.Range(func(x uint32) bool {
		array = append(array, x)
		sum += 1
		return true
	})
	return array[:sum]
}

// use for sameType operation
// example union,diffrence ...
type opSet interface {
	getLen() uint32
	getCap() uint32
	getMax() uint32
	load(int) uint32
	store(int, uint32)
}

// sameTypeUnion return set p of the Union set s and t
// item in s or in t.
//
// s,t,p must the same type
// time complexity: O(N/32)
func sameTypeUnion(s, t, p opSet) opSet {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
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

// sameTypeIntersect return set p of the Intersect set s and t
// item in s and in t.
//
// s,t,p must the same type
// time complexity: O(N/32)
func sameTypeIntersect(s, t, p opSet) opSet {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&t.load(i))
	}
	return p
}

// sameTypeDifference return set p of the Difference set s and t
// item in s but not in t.
//
// s,t,p must the same type
// time complexity: O(N/32)
func sameTypeDifference(s, t, p opSet) opSet {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	minLen := min(sLen, tLen)
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

// sameTypeComplement return set p of the Complement set s and t
// item in s but not in t and not in s but in t.
//
// s,t,p must the same type
// time complexity: O(N/32)
func sameTypeComplement(s, t, p opSet) opSet {
	sLen, tLen := int(s.getLen()), int(t.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
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

// sameTypeEqual return set s if equal t,
//
// s,t must the same type
// time complexity: O(N/32)
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

// sameTypeCopy copy s to t
// s.cap must <= t.cap
//
// time complexity: O(N/32)
func sameTypeCopy(s, t opSet) opSet {
	sLen := int(s.getLen())
	for i := 0; i < sLen; i++ {
		item := s.load(i)
		t.store(i, item)
	}
	return t
}

func items(s Set) []uint32 {
	var a = make([]uint32, 0, initSize)
	s.Range(func(x uint32) bool {
		a = append(a, x)
		return true
	})
	return a
}

func getItemsMaxMin(s, t Set) (ss, tt []uint32, maxcap, mincap int) {
	ss = make([]uint32, 0, initSize)
	tt = make([]uint32, 0, initSize)
	sm := uint32(0)
	s.Range(func(x uint32) bool {
		ss = append(ss, x)
		sm = x
		return true
	})
	tm := uint32(0)
	t.Range(func(x uint32) bool {
		tt = append(tt, x)
		tm = x
		return true
	})
	maxcap, mincap = maxmin(int(sm), int(tm))
	return ss, tt, maxcap, mincap
}

// generalUnion return set p of the Union set s and t
// item in s or in t.
// time complexity: O(N)
func generalUnion(s, t, p Set) Set {
	es, et, maxCap, _ := getItemsMaxMin(s, t)
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

// generalIntersect return set p of the Intersect set s and t
// item in s and in t.
// time complexity: O(N)
func generalIntersect(s, t, p Set) Set {
	es, et, _, minCap := getItemsMaxMin(s, t)
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

// generalDifference return set p of the Difference set s and t
// item in s but not in t.
// time complexity: O(N)
func generalDifference(s, t, p Set) Set {
	es, et, _, _ := getItemsMaxMin(s, t)
	maxCap := initSize
	if len(es) > 0 {
		maxCap = int(es[len(es)-1])
	}
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
			j += 1
			continue
		}
	}
	for ; i < len(es); i++ {
		p.Store(es[i])
	}
	return p
}

// generalComplement return set p of the Complement set s and t
// item in s but not in t and not in s but in t.
// time complexity: O(N)
func generalComplement(s, t, p Set) Set {
	es, et, maxCap, _ := getItemsMaxMin(s, t)
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

// generalEqual return set s if equal t,
// time complexity: O(N)
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

// generalCopy return a copy of s
//
// time complexity: O(N)
func generalCopy(s, t Set) Set {
	s.Range(func(x uint32) bool {
		t.Store(x)
		return true
	})
	return t
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

// ToStatic convert Set to Static set
// if set if Static,return ptr.
// or return a copy with set.
func ToStatic(s Set) *Static {
	rt := reflect.TypeOf(s)
	switch rt {
	case StaticType:
		return s.(*Static)
	case TrendsType:
		ss := s.(*Dynamic)
		return trendsToStatic(ss)
	// case OptionType:
	// 	ss := s.(*Option)
	// 	return ss.Static()
	default:
		array := items(s)
		slen := len(array)
		smax := array[slen-1]
		var ss Static
		ss.OnceInit(int(smax))
		for i := 0; i < slen; i++ {
			ss.Store(array[i])
		}
		return &ss
	}
}

// ToTrends convert Set to Trends set
// if set if Trends,return ptr.
// or return a copy with set.
func ToTrends(s Set) *Dynamic {
	rt := reflect.TypeOf(s)
	switch rt {
	case StaticType:
		ss := s.(*Static)
		return staticToTrends(ss)
	case TrendsType:
		return s.(*Dynamic)
	// case OptionType:
	// 	ss := s.(*Option)
	// 	return ss.Trends()
	default:
		array := items(s)
		slen := len(array)
		smax := array[slen-1]
		var ss Dynamic
		ss.OnceInit(int(smax))
		for i := 0; i < slen; i++ {
			ss.Store(array[i])
		}
		return &ss
	}
}

// trendsToStatic convert Trends set to Static set
func trendsToStatic(s *Dynamic) *Static {
	node := s.getEntry()
	var n Static
	// slen := int(node.getLen())
	n.onceInit(int(s.getMax()))
	u16To32(node, &n)
	// for i := 0; i < slen; i++ {
	// 	item := node.load(i)
	// 	// u32 实际存位值
	// 	ni := (i + 1) * 31 / 32
	// 	//有效补偿位
	// 	bit := i % 32

	// 	// ni - 补偿
	// 	ivalue := (item &^ (1<<bit - 1)) >> bit
	// 	n.store(ni, n.load(ni)|ivalue)

	// 	// 补偿ni-(i>>5+1)
	// 	if i%32 != 0 {
	// 		bvalue := item & (1<<bit - 1)
	// 		bvalue <<= 31 - ((i - 1) % 32)
	// 		n.store(i-(i>>5+1), n.load(i-(i>>5+1))|bvalue)
	// 	}
	// }
	return &n
}

// staticToTrends convert Static set to Trends set
func staticToTrends(s *Static) *Dynamic {
	var n Dynamic
	// slen := int(s.getLen())
	// nCap := (slen + slen/31) * 32
	smax := s.getMax()
	n.onceInit(int(smax))
	// n.max = uint32(maximum)
	// ncap := int(n.getEntry().cap)

	u32To16(s, n.getEntry())

	// for i := 0; i < slen; i++ {
	// 	item := s.load(i)
	// 	// u32 实际存位值
	// 	ni := i + i/31
	// 	//有效补偿位
	// 	bit := i % 31
	// 	ivalue := item << bit
	// 	// 去掉bit最高位
	// 	ivalue &^= 1 << 31
	// 	n.store(ni, n.load(ni)|ivalue)

	// 	if ni+1 < ncap {
	// 		// 补偿i+(i/31+1)
	// 		bv := item >> (31 - bit)
	// 		n.store(ni+1, n.load(ni+1)|bv)
	// 	}
	// }
	return &n
}

func u32To16(old, new opSet) {
	ocap := old.getCap()
	for i := 0; i < int(ocap); i++ {
		item := old.load(i)
		if item == 0 {
			continue
		}
		new.store(2*i, item&(1<<16-1))
		new.store(2*i+1, item>>16)
	}
}

func u16To32(old, new opSet) {
	ocap := old.getCap()
	for i := 0; i < int(ocap); i++ {
		item := old.load(i)
		item &^= freezeBit
		if item == 0 {
			continue
		}
		new.store(i>>1, new.load(i>>1)|item<<(16*(i&1)))
	}
}

// func u32To31(old, new opSet) {
// 	slen := int(old.getLen())
// 	ncap := int(new.getCap())
// 	for i := 0; i < slen; i++ {
// 		item := old.load(i)
// 		// u32 实际存位值
// 		ni := i + i/31
// 		//有效补偿位
// 		bit := i % 31
// 		ivalue := item << bit
// 		// 去掉bit最高位
// 		ivalue &^= freezeBit
// 		new.store(ni, new.load(ni)|ivalue)

// 		if ni+1 < ncap {
// 			// 补偿i+(i/31+1)
// 			bv := item >> (31 - bit)
// 			new.store(ni+1, new.load(ni+1)|bv)
// 		}
// 	}
// }

// func u31To32(old, new opSet) {
// 	slen := int(old.getLen())
// 	for i := 0; i < slen; i++ {
// 		item := old.load(i)
// 		// 去掉最高位
// 		item &^= freezeBit
// 		// u32 实际存位值
// 		ni := (i + 1) * 31 / 32
// 		//有效补偿位
// 		bit := i % 32

// 		// ni - 补偿
// 		ivalue := (item &^ (1<<bit - 1)) >> bit
// 		new.store(ni, new.load(ni)|ivalue)

// 		// 补偿ni-(i>>5+1)
// 		if i%32 != 0 {
// 			bvalue := item & (1<<bit - 1)
// 			bvalue <<= 31 - ((i - 1) % 32)
// 			oval := new.load(i - (i>>5 + 1))
// 			new.store(i-(i>>5+1), oval|bvalue)
// 		}
// 	}
// }

// // Static return a copy of option
// func (s *Option) Static() *Static {
// 	e := s.getEntry()
// 	maxcap := s.getMax()

// 	old := newOptEntry(maxcap, e.typ)
// 	sameTypeCopy(e, old)

// 	if old.typ != optEntry32 {
// 		ne := newOptEntry32(maxcap)
// 		optEvacuteEntry(old, ne, maxcap)
// 		old = ne
// 	}
// 	var p Static
// 	p.onceInit(int(s.getMax()))
// 	sameTypeCopy(old, &p)
// 	return &p
// }

// // Static return a copy of option
// func (s *Option) Trends() *Trends {
// 	e := s.getEntry()
// 	maxcap := s.getMax()

// 	old := newOptEntry(maxcap, e.typ)
// 	sameTypeCopy(e, old)

// 	if old.typ != optEntry31 {
// 		ne := newOptEntry31(maxcap)
// 		optEvacuteEntry(old, ne, maxcap)
// 		old = ne
// 	}
// 	var p Trends
// 	p.onceInit(int(s.getMax()))
// 	sameTypeCopy(old, &p)
// 	return &p
// }

// Adds Store all x in args to the set
func Adds(s Set, args ...uint32) {
	for _, x := range args {
		s.Store(x)
	}
}

// Removes Delete all x in args from the set
func Removes(s Set, args ...uint32) {
	for _, x := range args {
		s.Delete(x)
	}
}

// Null report set if empty
//
// time complexity: O(N/32)
func Null(s Set) bool {
	var flag = true
	s.Range(func(x uint32) bool {
		flag = false
		return false
	})
	return flag
}

// Size return the number of elements in set
func Size(s Set) int {
	r := reflect.TypeOf(s)
	var size uint32
	switch r {
	case StaticType:
		ss := s.(*Static)
		size = atomic.LoadUint32(&ss.count)
		if size == 0 {
			ss.Range(func(x uint32) bool {
				size += 1
				return true
			})
			atomic.CompareAndSwapUint32(&ss.count, 0, size)
		}
	case TrendsType:
		ss := s.(*Dynamic)
		e := ss.getEntry()
		size = atomic.LoadUint32(&e.count)
		if size == 0 {
			ss.Range(func(x uint32) bool {
				size += 1
				return true
			})
			atomic.CompareAndSwapUint32(&e.count, 0, size)
		}
	// case OptionType:
	// 	ss := s.(*Option)
	// 	e := ss.getEntry()
	// 	size = e.getCount()
	// 	if size == 0 {
	// 		ss.Range(func(x uint32) bool {
	// 			size += 1
	// 			return true
	// 		})
	// 		atomic.CompareAndSwapUint32(&e.count, 0, size)
	// 	}
	default:
		s.Range(func(x uint32) bool {
			size += 1
			return true
		})
	}
	return int(size)
}

// Clear remove all elements from the set
// time complexity: O(N/32)
func Clear(s Set) {
	r := reflect.TypeOf(s)
	switch r {
	case StaticType:
		ss := s.(*Static)
		slen := ss.getLen()
		for i := 0; i < int(slen); i++ {
			ss.store(i, 0)
		}
		atomic.StoreUint32(&ss.count, 0)
		atomic.CompareAndSwapUint32(&ss.len, slen, 0)
	case TrendsType:
		ss := s.(*Dynamic)
		for {
			n := ss.getEntry()
			ne := newNode(ss.getMax())
			if atomic.CompareAndSwapPointer(&ss.node, unsafe.Pointer(n), unsafe.Pointer(ne)) {
				break
			}
		}
	// case OptionType:
	// 	ss := s.(*Option)
	// 	for {
	// 		e := ss.getEntry()
	// 		ne := newOptEntry(ss.getMax(), e.typ)
	// 		if atomic.CompareAndSwapPointer(&ss.node, unsafe.Pointer(e), unsafe.Pointer(ne)) {
	// 			break
	// 		}
	// 	}
	default:
		s.Range(func(x uint32) bool {
			s.Delete(x)
			return true
		})
	}
}
