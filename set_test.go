package set_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/min1324/set"
)

type setOp string

const (
	opLoad          = setOp("Load")
	opStore         = setOp("Store")
	opDelete        = setOp("Delete")
	opLoadOrStore   = setOp("LoadOrStore")
	opLoadAndDelete = setOp("LoadAndDelete")
	opRange         = setOp("Range")
	opLen           = setOp("Len")
	opClear         = setOp("Clear")
	opCopy          = setOp("Copy")
	opNull          = setOp("Null")
	opItems         = setOp("Items")
)

var setOps = [...]setOp{opLoad, opLoadOrStore, opLoadAndDelete}

// setCall is a quick.Generator for calls on mapInterface.
type setCall struct {
	op setOp
	k  uint32
}

func (c setCall) apply(m Interface) (uint32, bool) {
	switch c.op {
	case opLoad:
		return c.k, m.Load(c.k)
	case opLoadOrStore:
		l, _ := m.LoadOrStore(c.k)
		return c.k, l
	case opLoadAndDelete:
		l, _ := m.LoadAndDelete(c.k)
		return c.k, l
	default:
		panic("invalid mapOp")
	}
}

type setResult struct {
	value uint32
	ok    bool
}

func randValue(r *rand.Rand) uint32 {
	return uint32(rand.Int31n(32 << 5))
}

func (setCall) Generate(r *rand.Rand, size int) reflect.Value {
	c := setCall{op: setOps[rand.Intn(len(setOps))], k: randValue(r)}
	return reflect.ValueOf(c)
}

func applyCalls(m Interface, calls []setCall) (results []setResult, final map[interface{}]interface{}) {
	for _, c := range calls {
		v, ok := c.apply(m)
		results = append(results, setResult{v, ok})
	}

	final = make(map[interface{}]interface{})
	m.Range(func(x uint32) bool {
		final[x] = true
		return true
	})

	return results, final
}

func applyIntSet(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(set.IntSet), calls)
}

func applySliceSet(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(set.IntSet), calls)
}

func applyMutex(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(MutexSet), calls)
}

func TestIntSetMatchsSlice(t *testing.T) {
	if err := quick.CheckEqual(applyIntSet, applySliceSet, nil); err != nil {
		t.Error(err)
	}
}

func TestIntSetMatchsMutex(t *testing.T) {
	if err := quick.CheckEqual(applyIntSet, applyMutex, nil); err != nil {
		t.Error(err)
	}
}

func TestSliceSetMatchsMutex(t *testing.T) {
	if err := quick.CheckEqual(applySliceSet, applyMutex, nil); err != nil {
		t.Error(err)
	}
}

func getIntSet(cap, m, n int) *set.IntSet {
	var s set.IntSet
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

func getSliceSet(cap, m, n int) *set.SliceSet {
	var s set.SliceSet
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

func getMutexSet(cap, m, n int) *MutexSet {
	var s MutexSet
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

type setStruct struct {
	setup func(*testing.T, Interface)
	run   func(*testing.T, Interface)
}

const (
	initCap = 100
	initM   = 0
	initN   = 36
)

func queueMap(t *testing.T, test setStruct) {
	for _, m := range [...]Interface{
		&set.IntSet{},
		&set.SliceSet{},
		&MutexSet{},
	} {
		t.Run(fmt.Sprintf("%T", m), func(t *testing.T) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(Interface)
			if test.setup != nil {
				test.setup(t, m)
			}
			test.run(t, m)
		})
	}
}

func TestInit(t *testing.T) {
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			if !s.Load(0) {
				t.Fatalf("load exist err:%d", 0)
			}

			if !s.Load(31) {
				t.Fatalf("load exist err:%d", 31)
			}

			if !s.Load(35) {
				t.Fatalf("load exist err:%d", 35)
			}

			if s.Load(36) {
				t.Fatalf("load not exist err:%d", 36)
			}

			s.Store(55)
			if !s.Load(55) {
				t.Fatalf("Store exist err:%d", 55)
			}

			s.Store(101)
			if s.Load(101) {
				t.Fatalf("Store overflow err:%d", 101)
			}

			s.Delete(55)
			if s.Load(55) {
				t.Fatalf("delete exist err:%d", 55)
			}
		},
	})
}

func TestRange(t *testing.T) {
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			count := 0
			s.Range(func(x uint32) bool {
				if x != uint32(count) {
					t.Fatalf("range err need:%d,real:%d", count, x)
				}
				count += 1
				return true
			})
		},
	})
}

func TestLen(t *testing.T) {
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			slen := s.Len()
			if s.Len() != initN {
				t.Fatalf("len err:%d,need:%d", slen, initN)
			}
			for i := 10; i < 20; i++ {
				s.Delete(uint32(i))
			}
			slen = s.Len()
			if slen != initN-10 {
				t.Fatalf("len err:%d,need:%d", slen, initN-10)
			}
		},
	})
}

func TestClear(t *testing.T) {
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			s.Clear()
			slen := s.Len()
			if slen != 0 {
				t.Errorf("Clear err,%d!=0", slen)
			}
			s.Range(func(x uint32) bool {
				t.Fatalf("Clear not empty:%d", x)
				return true
			})
		},
	})
}

func TestNull(t *testing.T) {
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
		},
		run: func(t *testing.T, s Interface) {
			if !s.Null() {
				t.Fatalf("init Null err")
			}
			s.Store(1)
			s.Store(2)
			s.Store(3)
			if s.Null() {
				t.Fatalf("Adds not Null err")
			}
			s.Delete(1)
			s.Delete(2)
			s.Delete(3)
			if !s.Null() {
				t.Fatalf("Removes not Null err")
			}
		},
	})
}

func TestItems(t *testing.T) {
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			array := s.Items()
			slen := s.Len()
			if slen != len(array) {
				t.Fatalf("items len err:%d,%d", slen, len(array))
			}
			var result = make(map[uint32]bool)
			s.Range(func(x uint32) bool {
				result[x] = true
				return true
			})
			for _, v := range array {
				result[v] = false
			}
			for k, v := range result {
				if v {
					t.Fatalf("items miss:%d", k)
				}
			}
		},
	})
}

func TestEqual(t *testing.T) {
	q := getIntSet(initCap, initM, initN)
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			if !set.Equal(s, s) {
				t.Fatalf("Equal err, s!=s")
			}
			if !set.Equal(s, q) {
				t.Fatalf("Equal err, s!=q")
			}
		},
	})
}

func TestUnion(t *testing.T) {
	iq := getIntSet(10, 2, 8)
	ir := getIntSet(10, 0, 8)
	ie := getIntSet(10, 0, 8)

	sq := getSliceSet(10, 2, 8)
	sr := getSliceSet(10, 0, 8)
	se := getIntSet(10, 0, 8)

	mq := getMutexSet(10, 2, 8)
	mr := getMutexSet(10, 0, 8)
	me := getIntSet(10, 0, 8)
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(10)
			for i := 0; i < 5; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			ip := set.Union(s, iq)
			if !set.Equal(ip, ir) {
				t.Fatalf("union err:%v,%v", ir, ip)
			}
			io := set.Union(iq, s)
			if !set.Equal(io, ie) {
				t.Fatalf("union err:%v,%v", ie, io)
			}

			sp := set.Union(s, sq)
			if !set.Equal(sp, sr) {
				t.Fatalf("union err:%v,%v", sr, sp)
			}
			so := set.Union(sq, s)
			if !set.Equal(so, se) {
				t.Fatalf("union err:%v,%v", se, so)
			}

			mp := set.Union(s, mq)
			if !set.Equal(mp, mr) {
				t.Fatalf("union err:%v,%v", mr, mp)
			}
			mo := set.Union(mq, s)
			if !set.Equal(mo, me) {
				t.Fatalf("union err:%v,%v", me, mo)
			}

		},
	})
}

func TestIntersect(t *testing.T) {
	iq := getIntSet(10, 2, 8)
	ir := getIntSet(10, 2, 5)
	ie := getIntSet(10, 2, 5)

	sq := getSliceSet(10, 2, 8)
	sr := getSliceSet(10, 2, 5)
	se := getSliceSet(10, 2, 5)

	mq := getMutexSet(10, 2, 8)
	mr := getMutexSet(10, 2, 5)
	me := getMutexSet(10, 2, 5)
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(10)
			for i := 0; i < 5; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			ip := set.Intersect(s, iq)
			if !set.Equal(ip, ir) {
				t.Fatalf("Intersect err:%v,%v", ip, ir)
			}
			io := set.Intersect(iq, s)
			if !set.Equal(io, ie) {
				t.Fatalf("Intersect err:%v,%v", ie, io)
			}

			sp := set.Intersect(s, sq)
			if !set.Equal(sp, sr) {
				t.Fatalf("Intersect err:%v,%v", sp, sr)
			}
			so := set.Intersect(sq, s)
			if !set.Equal(so, se) {
				t.Fatalf("Intersect err:%v,%v", se, so)
			}

			mp := set.Intersect(s, mq)
			if !set.Equal(mp, mr) {
				t.Fatalf("Intersect err:%v,%v", mp, mr)
			}
			mo := set.Intersect(mq, s)
			if !set.Equal(mo, me) {
				t.Fatalf("Intersect err:%v,%v", me, mo)
			}
		},
	})
}

func TestDifference(t *testing.T) {
	iq := getIntSet(10, 2, 8)
	ir := getIntSet(10, 0, 2)
	ie := getIntSet(10, 5, 8)

	sq := getSliceSet(10, 2, 8)
	sr := getSliceSet(10, 0, 2)
	se := getIntSet(10, 5, 8)

	mq := getMutexSet(10, 2, 8)
	mr := getMutexSet(10, 0, 2)
	me := getIntSet(10, 5, 8)

	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(10)
			for i := 0; i < 5; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			ip := set.Difference(s, iq)
			if !set.Equal(ip, ir) {
				t.Fatalf("Difference err:%v,%v", ir, ip)
			}
			io := set.Difference(iq, s)
			if !set.Equal(io, ie) {
				t.Fatalf("Difference err:%v,%v", ie, io)
			}

			sp := set.Difference(s, sq)
			if !set.Equal(sp, sr) {
				t.Fatalf("Difference err:%v,%v", sr, sp)
			}
			so := set.Difference(sq, s)
			if !set.Equal(so, se) {
				t.Fatalf("Difference err:%v,%v", se, so)
			}

			mp := set.Difference(s, mq)
			if !set.Equal(mp, mr) {
				t.Fatalf("Difference err:%v,%v", mr, mp)
			}
			mo := set.Difference(mq, s)
			if !set.Equal(mo, me) {
				t.Fatalf("Difference err:%v,%v", me, mo)
			}
		},
	})
}

func TestComplement(t *testing.T) {
	iq := getIntSet(10, 2, 8)
	ir := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))
	ie := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))

	sq := getIntSet(10, 2, 8)
	sr := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))
	se := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))

	mq := getIntSet(10, 2, 8)
	mr := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))
	me := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(10)
			for i := 0; i < 5; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			ip := set.Complement(s, iq)
			if !set.Equal(ip, ir) {
				t.Fatalf("Complement err:%v,%v", ir, ip)
			}
			io := set.Complement(iq, s)
			if !set.Equal(io, ie) {
				t.Fatalf("Complement err:%v,%v", ie, io)
			}

			sp := set.Complement(s, sq)
			if !set.Equal(sp, sr) {
				t.Fatalf("Complement err:%v,%v", sr, sp)
			}
			so := set.Complement(sq, s)
			if !set.Equal(so, se) {
				t.Fatalf("Complement err:%v,%v", se, so)
			}

			mp := set.Complement(s, mq)
			if !set.Equal(mp, mr) {
				t.Fatalf("Complement err:%v,%v", mr, mp)
			}
			mo := set.Complement(mq, s)
			if !set.Equal(mo, me) {
				t.Fatalf("Complement err:%v,%v", me, mo)
			}
		},
	})
}

var raceOps = [...]setOp{
	opLoad,
	opStore,
	opDelete,
	opLoadOrStore,
	opLoadAndDelete,
	opRange,
	opLen,
	opClear,
	opNull,
	opItems,
}

func (c setCall) raceCall(s Interface) {
	switch c.op {
	case opLoad:
		s.Load(c.k)
	case opStore:
		s.Store(c.k)
	case opDelete:
		s.Delete(c.k)
	case opLoadOrStore:
		s.LoadOrStore(c.k)
	case opLoadAndDelete:
		s.LoadAndDelete(c.k)
	case opRange:
		s.Range(func(x uint32) bool { return true })
	case opLen:
		s.Len()
	case opClear:
		s.Clear()
	case opNull:
		s.Null()
	case opItems:
		s.Items()
	default:
		panic("invalid mapOp")
	}
}

func generate(r *rand.Rand) *setCall {
	return &setCall{op: setOps[rand.Intn(len(setOps))], k: randValue(r)}
}

func TestRace(t *testing.T) {
	var goNum = runtime.NumCPU()
	var wg sync.WaitGroup
	var r = rand.New(rand.NewSource(time.Now().Unix()))

	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			wg.Add(goNum)
			for i := 0; i < goNum; i++ {
				go func() {
					defer wg.Done()
					for i := 0; i < 100000; i++ {
						generate(r).raceCall(s)
					}
				}()
			}
			wg.Done()
		},
	})
}
