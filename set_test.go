package set_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
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
		panic("invalid setOp")
	}
}

type setResult struct {
	value uint32
	ok    bool
}

func randValue(r *rand.Rand) uint32 {
	return uint32(rand.Int31n(32 << 4))
}

func (setCall) Generate(r *rand.Rand, size int) reflect.Value {
	c := setCall{op: setOps[rand.Intn(len(setOps))], k: randValue(r)}
	return reflect.ValueOf(c)
}

func applyCalls(m Interface, calls []setCall) (results []setResult, final map[interface{}]interface{}) {
	m.OnceInit(int(maxItem))
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

func applyFixed(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(set.Option), calls)
}

func applyFixedInt(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewIntOpt(int(maxItem)), calls)
}

func applyFixedVar(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewVarOpt(int(maxItem)), calls)
}

type applyFunc func(calls []setCall) ([]setResult, map[interface{}]interface{})

type applyStruct struct {
	name string
	applyFunc
}

func applyMap(t *testing.T, standard applyFunc) {
	for _, m := range [...]applyStruct{
		{"IntSet", applyIntSet},
		{"SliceSet", applySliceSet},
		{"Mutex", applyMutex},
		{"Fixed", applyFixed},
		{"FixedInt", applyFixedInt},
		{"FixedVar", applyFixedVar},
	} {
		t.Run(m.name, func(t *testing.T) {
			if err := quick.CheckEqual(standard, m.applyFunc, nil); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAll(t *testing.T) {
	applyMap(t, applyMutex)
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

func TestMutexMatchsFixed(t *testing.T) {
	if err := quick.CheckEqual(applyMutex, applyFixed, nil); err != nil {
		t.Error(err)
	}
}

func TestMutexMatchsFixedInt(t *testing.T) {
	if err := quick.CheckEqual(applyMutex, applyFixedInt, nil); err != nil {
		t.Error(err)
	}
}

func TestIntSetMatchsFixedVar(t *testing.T) {
	if err := quick.CheckEqual(applyMutex, applyFixedVar, nil); err != nil {
		t.Error(err)
	}
}

func TestFixedMatchsFixedVar(t *testing.T) {
	if err := quick.CheckEqual(applyFixedInt, applyFixedVar, nil); err != nil {
		t.Error(err)
	}
}

func TestSliceMatchsFixedVar(t *testing.T) {
	if err := quick.CheckEqual(applySliceSet, applyFixedVar, nil); err != nil {
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

func getFixedSet(cap, m, n int) *set.Option {
	var s set.Option
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
		set.NewIntOpt(preInitSize),
		set.NewVarOpt(preInitSize),
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
			s.Delete(35)
			if s.Load(35) {
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
	iq := getIntSet(initCap, initM, initN)
	sq := getIntSet(initCap, initM, initN)
	mq := getIntSet(initCap, initM, initN)
	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(initCap)
			for i := initM; i < initN; i++ {
				s.Store(uint32(i))
			}
		},
		run: func(t *testing.T, s Interface) {
			if !set.Equal(s, s) {
				t.Errorf("Equal err, s!=s")
			}
			if !set.Equal(s, iq) {
				t.Errorf("Equal Int err, s!=iq")
			}
			if !set.Equal(s, sq) {
				t.Errorf("Equal Slice err, s!=sq")
			}
			if !set.Equal(s, mq) {
				t.Errorf("Equal Mutex err, s!=mq")
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
	se := getSliceSet(10, 0, 8)

	mq := getMutexSet(10, 2, 8)
	mr := getMutexSet(10, 0, 8)
	me := getMutexSet(10, 0, 8)
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
				t.Errorf("union Int err:%v,%v", ir, ip)
			}
			io := set.Union(iq, s)
			if !set.Equal(io, ie) {
				t.Errorf("union Int err:%v,%v", ie, io)
			}

			sp := set.Union(s, sq)
			if !set.Equal(sp, sr) {
				t.Errorf("union Slice err:%v,%v", sr, sp)
			}
			so := set.Union(sq, s)
			if !set.Equal(so, se) {
				t.Errorf("union Slice err:%v,%v", se, so)
			}

			mp := set.Union(s, mq)
			if !set.Equal(mp, mr) {
				t.Errorf("union Mutex err:%v,%v", mr, mp)
			}
			mo := set.Union(mq, s)
			if !set.Equal(mo, me) {
				t.Errorf("union Mutex err:%v,%v", me, mo)
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
				t.Errorf("Intersect Int err:%v,%v", ip, ir)
			}
			io := set.Intersect(iq, s)
			if !set.Equal(io, ie) {
				t.Errorf("Intersect Int err:%v,%v", ie, io)
			}

			sp := set.Intersect(s, sq)
			if !set.Equal(sp, sr) {
				t.Errorf("Intersect Slice err:%v,%v", sp, sr)
			}
			so := set.Intersect(sq, s)
			if !set.Equal(so, se) {
				t.Errorf("Intersect Slice err:%v,%v", se, so)
			}

			mp := set.Intersect(s, mq)
			if !set.Equal(mp, mr) {
				t.Errorf("Intersect Mutex err:%v,%v", mp, mr)
			}
			mo := set.Intersect(mq, s)
			if !set.Equal(mo, me) {
				t.Errorf("Intersect Mutex err:%v,%v", me, mo)
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
	se := getSliceSet(10, 5, 8)

	mq := getMutexSet(10, 2, 8)
	mr := getMutexSet(10, 0, 2)
	me := getMutexSet(10, 5, 8)

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
				t.Errorf("Difference Int err:%v,%v", ir, ip)
			}
			io := set.Difference(iq, s)
			if !set.Equal(io, ie) {
				t.Errorf("Difference Int err:%v,%v", ie, io)
			}

			sp := set.Difference(s, sq)
			if !set.Equal(sp, sr) {
				t.Errorf("Difference Slice err:%v,%v", sr, sp)
			}
			so := set.Difference(sq, s)
			if !set.Equal(so, se) {
				t.Errorf("Difference Slice err:%v,%v", se, so)
			}

			mp := set.Difference(s, mq)
			if !set.Equal(mp, mr) {
				t.Errorf("Difference Mutex err:%v,%v", mr, mp)
			}
			mo := set.Difference(mq, s)
			if !set.Equal(mo, me) {
				t.Errorf("Difference Mutex err:%v,%v", me, mo)
			}
		},
	})
}

func TestComplement(t *testing.T) {
	iq := getIntSet(10, 2, 8)
	ir := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))
	ie := set.Union(getIntSet(10, 0, 2), getIntSet(10, 5, 8))

	sq := getSliceSet(10, 2, 8)
	sr := set.Union(getSliceSet(10, 0, 2), getSliceSet(10, 5, 8))
	se := set.Union(getSliceSet(10, 0, 2), getSliceSet(10, 5, 8))

	mq := getMutexSet(10, 2, 8)
	mr := set.Union(getMutexSet(10, 0, 2), getMutexSet(10, 5, 8))
	me := set.Union(getMutexSet(10, 0, 2), getMutexSet(10, 5, 8))
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
				t.Errorf("Complement Int err:%v,%v", ir, ip)
			}
			io := set.Complement(iq, s)
			if !set.Equal(io, ie) {
				t.Errorf("Complement Int err:%v,%v", ie, io)
			}

			sp := set.Complement(s, sq)
			if !set.Equal(sp, sr) {
				t.Errorf("Complement Slice err:%v,%v", sr, sp)
			}
			so := set.Complement(sq, s)
			if !set.Equal(so, se) {
				t.Errorf("Complement Slice err:%v,%v", se, so)
			}

			mp := set.Complement(s, mq)
			if !set.Equal(mp, mr) {
				t.Errorf("Complement Mutex err:%v,%v", mr, mp)
			}
			mo := set.Complement(mq, s)
			if !set.Equal(mo, me) {
				t.Errorf("Complement Mutex err:%v,%v", me, mo)
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
		panic("invalid mapOp:" + c.op)
	}
}

func TestRace(t *testing.T) {
	var goNum = runtime.NumCPU()
	var wg sync.WaitGroup
	var r = rand.New(rand.NewSource(time.Now().Unix()))
	var max = 10000

	args := make([]setCall, goNum*max)
	for i := range args {
		args[i].k, args[i].op = randValue(r), raceOps[rand.Intn(len(raceOps))]
	}

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
				j := i
				go func(j int) {
					defer wg.Done()
					for i := 0; i < max; i++ {
						args[i+j*max].raceCall(s)
					}
				}(j)
			}
			wg.Wait()
		},
	})
}

func TestConcurrentStore(t *testing.T) {
	var wg sync.WaitGroup
	goNum := runtime.NumCPU()
	var max = 10000

	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(max)
		},
		run: func(t *testing.T, s Interface) {
			// enqueue
			wg.Add(goNum)
			var gbCount uint32 = 0
			for i := 0; i < goNum; i++ {
				go func() {
					defer wg.Done()
					for {
						c := atomic.AddUint32(&gbCount, 1)
						if c >= uint32(max) {
							break
						}
						s.Store(c)
					}
				}()
			}
			// wait until finish
			wg.Wait()

			var count uint32 = 1
			// check
			s.Range(func(x uint32) bool {
				if x != count {
					// t.Fatalf("store err need:%d,real:%d", count, x)
					return false
				}
				count += 1
				return true
			})
		},
	})
}

func TestConcurrentDelete(t *testing.T) {
	var wg sync.WaitGroup
	goNum := runtime.NumCPU()
	var max = 10000

	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(max)
			for i := 1; i < max; i++ {
				if !s.Store(uint32(i)) {
					t.Fatalf("store err:%d", i)
				}
			}
		},
		run: func(t *testing.T, s Interface) {
			// enqueue
			wg.Add(goNum)
			var gbCount uint32 = 0
			for i := 0; i < goNum; i++ {
				go func() {
					defer wg.Done()
					for {
						c := atomic.AddUint32(&gbCount, 1)
						if c >= uint32(max) {
							break
						}
						s.Delete(c)
					}
				}()
			}
			// wait until finish
			wg.Wait()

			// check
			s.Range(func(x uint32) bool {
				t.Fatalf("delete err:%d", x)
				return false
			})
		},
	})
}

func TestConcurrentRace(t *testing.T) {
	var wg sync.WaitGroup
	goNum := runtime.NumCPU()
	var max = 10000

	r := rand.New(rand.NewSource(time.Now().Unix()))
	delargs := make([]uint32, max)
	strargs := make([]uint32, max)
	for i := 0; i < max; i++ {
		delargs[i] = uint32(r.Int63n(int64(max)))
		strargs[i] = uint32(r.Int63n(int64(max)))
	}

	queueMap(t, setStruct{
		setup: func(t *testing.T, s Interface) {
			s.OnceInit(max)
		},
		run: func(t *testing.T, s Interface) {
			// delete
			wg.Add(goNum)
			var delCount uint32 = 0
			for i := 0; i < goNum; i++ {
				go func() {
					defer wg.Done()
					for {
						c := atomic.AddUint32(&delCount, 1)
						if c >= uint32(max) {
							break
						}
						s.Delete(atomic.LoadUint32(&delargs[c]))
					}
				}()
			}
			// delete
			wg.Add(goNum)
			var strCount uint32 = 0
			for i := 0; i < goNum; i++ {
				go func() {
					defer wg.Done()
					for {
						c := atomic.AddUint32(&strCount, 1)
						if c >= uint32(max) {
							break
						}
						s.Store(atomic.LoadUint32(&strargs[c]))
					}
				}()
			}
			// wait until finish
			wg.Wait()

		},
	})
}

func Test_uint31To32(t *testing.T) {
	ulen := 100
	u31 := make([]uint32, ulen)
	u32 := make([]uint32, ulen)
	for i := 0; i < ulen; i++ {
		u31[i] |= 1<<31 - 1
		u32[i] |= 1<<31 - 1 | 1<<31
	}
	ls := ulen*31/32 + 1
	lm := ulen * 31 % 32
	u32 = u32[:ls]
	u32[ls-1] &= (1<<lm - 1)
	for i := ls; i < len(u32); i++ {
		u32[i] = 0
	}

	type args struct {
		u31  []uint32
		ulen int
	}
	tests := []struct {
		name    string
		args    args
		wantU32 []uint32
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				u31:  u31,
				ulen: ulen,
			},
			wantU32: u32,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotU32 := uint31To32(tt.args.u31, tt.args.ulen); !reflect.DeepEqual(gotU32, tt.wantU32) {
				for i := 0; i < len(gotU32); i++ {
					if gotU32[i] != tt.wantU32[i] {
						t.Errorf("uint31To32() = %v, want %v\n", gotU32[i], tt.wantU32[i])
					}
				}
			}
		})
	}
}

func uint31To32(u31 []uint32, ulen int) (u32 []uint32) {
	u32 = make([]uint32, ulen)
	mlen := ulen*31/32 + 1
	for i := 0; i < ulen; i++ {
		item := u31[i]
		ni := (i + 1) * 31 / 32
		bit := i % 32 //有效补偿位

		u32[ni] |= (item &^ (1<<bit - 1)) >> bit

		if i%32 != 0 {
			// 补偿
			bv := item & (1<<bit - 1)
			bv <<= 31 - ((i - 1) % 32)
			u32[i-(i/32+1)] |= bv
		}
	}
	return u32[:mlen]
}

func TestSliceToInt(t *testing.T) {
	cap := 1000
	var s set.SliceSet
	s.OnceInit(cap)
	for i := 0; i < cap; i++ {
		s.Store(uint32(i))
	}
	r := s.Copy()
	type args struct {
		s *set.SliceSet
	}
	tests := []struct {
		name string
		args args
		want *set.IntSet
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				s: &s,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.SliceToInt(tt.args.s); !set.Equal(got, r) {
				// t.Errorf("SliceToInt() = %v, want %v", got, r)
				var i uint32 = 0
				got.Range(func(x uint32) bool {
					if i != x {
						t.Errorf("miss:%d ", i)
					}
					i = x + 1
					return true
				})
			}
		})
	}
}

func TestIntToSlice(t *testing.T) {
	cap := 1000
	var s set.IntSet
	s.OnceInit(cap)
	for i := 0; i < cap; i++ {
		s.Store(uint32(i))
	}
	r := s.Copy()
	type args struct {
		s *set.IntSet
	}
	tests := []struct {
		name string
		args args
		want *set.SliceSet
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				s: &s,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.IntToSlice(tt.args.s); !set.Equal(got, r) {
				// t.Errorf("SliceToInt() = %v, want %v", got, r)
				var i uint32 = 0
				got.Range(func(x uint32) bool {
					if i != x {
						t.Errorf("miss:%d ", i)
					}
					i = x + 1
					return true
				})
			}
		})
	}
}
