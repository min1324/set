package set_test

import (
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

func (c setCall) apply(m setInterface) (uint32, bool) {
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
	return uint32(rand.Int31n(32 * 5))
}

func (setCall) Generate(r *rand.Rand, size int) reflect.Value {
	c := setCall{op: setOps[rand.Intn(len(setOps))], k: randValue(r)}
	return reflect.ValueOf(c)
}

func applyCalls(m setInterface, calls []setCall) (results []setResult, final map[interface{}]interface{}) {
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

func applySet(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.New(), calls)
}

func applyMutex(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(MutexSet), calls)
}

func TestMatchesSet(t *testing.T) {
	if err := quick.CheckEqual(applySet, applySet, nil); err != nil {
		t.Error(err)
	}
}

func TestMatchesMutex(t *testing.T) {
	if err := quick.CheckEqual(applySet, applyMutex, nil); err != nil {
		t.Error(err)
	}
}

func initSet(n int) *set.IntSet {
	var s set.IntSet
	for i := 0; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

func TestInit(t *testing.T) {
	s := initSet(2)
	if !s.Load(0) {
		t.Fatalf("load exist err:%d", 0)
	}

	if s.Load(3) {
		t.Fatalf("load not exist err:%d", 3)
	}

	if s.Load(33) {
		t.Fatalf("load not exist err:%d", 33)
	}
	s.Store(33)
	if !s.Load(33) {
		t.Fatalf("Store exist err:%d", 33)
	}
	s.Delete(33)
	if s.Load(33) {
		t.Fatalf("delete exist err:%d", 33)
	}
}

func TestRange(t *testing.T) {
	s := initSet(100)
	count := 0
	s.Range(func(x uint32) bool {
		if x != uint32(count) {
			t.Fatalf("range err need:%d,real:%d", count, x)
		}
		count += 1
		return true
	})
}

func TestLen(t *testing.T) {
	s := initSet(100)
	if s.Len() != 100 {
		t.Fatalf("len err")
	}
	for i := 10; i < 30; i++ {
		s.Delete(uint32(i))
	}
	slen := s.Len()
	if slen != 80 {
		t.Fatalf("len err:%d", slen)
	}
}

func TestClear(t *testing.T) {
	s := initSet(10)
	s.Clear()
	if s.Len() != 0 {
		t.Fatalf("Clear err,len!=0")
	}
	s.Range(func(x uint32) bool {
		t.Fatalf("Clear not empty")
		return true
	})
}

func TestNull(t *testing.T) {
	var s set.IntSet
	if !s.Null() {
		t.Fatalf("init Null err")
	}
	s.Adds(1, 2, 3)
	if s.Null() {
		t.Fatalf("Adds not Null err")
	}
	s.Removes(1, 2, 3)
	if !s.Null() {
		t.Fatalf("Removes not Null err")
	}
}

func TestEqual(t *testing.T) {
	s := initSet(10)
	if !set.Equal(s, s) {
		t.Fatalf("Equal err, s!=s")
	}
	p := initSet(10)
	if !set.Equal(s, p) {
		t.Fatalf("Equal err, s!=p")
	}
}

func TestItems(t *testing.T) {
	s := initSet(10)
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
}

func TestCopy(t *testing.T) {
	s := initSet(10)
	p := s.Copy()
	if !set.Equal(s, p) {
		t.Fatalf("Copy err, s!=p")
	}
}

// return [m,n)
func initSetR(m, n int) *set.IntSet {
	var s set.IntSet
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

func TestUnion(t *testing.T) {
	s := initSet(10)
	r := initSetR(10, 36)

	p := initSet(36)
	q := set.Union(s, r)

	if !set.Equal(p, q) {
		t.Logf("S:%v", s.Items())
		t.Logf("R:%v", r.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", q.Items())
		t.Fatalf("Union err")
	}
	k := initSet(10)
	j := initSetR(10, 36)
	l := set.Union(j, k)
	if !set.Equal(p, l) {
		t.Logf("S:%v", j.Items())
		t.Logf("R:%v", k.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", l.Items())
		t.Fatalf("Union err")
	}
}

func TestIntersect(t *testing.T) {
	s := initSet(20)
	r := initSetR(10, 36)

	p := initSetR(10, 20)
	q := set.Intersect(s, r)

	if !set.Equal(p, q) {
		t.Logf("S:%v", s.Items())
		t.Logf("R:%v", r.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", q.Items())
		t.Fatalf("Intersect err")
	}

	k := initSet(20)
	j := initSetR(10, 36)
	l := set.Intersect(j, k)
	if !set.Equal(p, l) {
		t.Logf("S:%v", j.Items())
		t.Logf("R:%v", k.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", l.Items())
		t.Fatalf("Intersect err")
	}
}

func TestDifference(t *testing.T) {
	s := initSet(20)
	r := initSetR(10, 36)

	p := initSet(10)
	q := set.Difference(s, r)

	if !set.Equal(p, q) {
		t.Logf("S:%v", s.Items())
		t.Logf("R:%v", r.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", q.Items())
		t.Fatalf("Difference err")
	}

	k := initSet(20)
	j := initSetR(10, 36)
	h := initSetR(20, 36)
	l := set.Difference(j, k)
	if !set.Equal(h, l) {
		t.Logf("S:%v", j.Items())
		t.Logf("R:%v", k.Items())
		t.Logf("P:%v", h.Items())
		t.Logf("Q:%v", l.Items())
		t.Fatalf("Difference err")
	}
}

func TestComplement(t *testing.T) {
	s := initSet(20)
	r := initSetR(10, 36)

	p := set.Union(initSet(10), initSetR(20, 36))
	q := set.Complement(s, r)

	if !set.Equal(p, q) {
		t.Logf("S:%v", s.Items())
		t.Logf("R:%v", r.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", q.Items())
		t.Fatalf("Complement err")
	}

	k := initSet(20)
	j := initSetR(10, 36)
	l := set.Complement(j, k)
	if !set.Equal(p, l) {
		t.Logf("S:%v", j.Items())
		t.Logf("R:%v", k.Items())
		t.Logf("P:%v", p.Items())
		t.Logf("Q:%v", l.Items())
		t.Fatalf("Complement err")
	}
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
	opCopy,
	opNull,
	opItems,
}

func (c setCall) raceCall(s *set.IntSet) {
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
	case opCopy:
		s.Copy()
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
	var s set.IntSet
	var r = rand.New(rand.NewSource(time.Now().Unix()))
	wg.Add(goNum)
	for i := 0; i < goNum; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 100000; i++ {
				generate(r).raceCall(&s)
			}
		}()
	}
	wg.Done()
}
