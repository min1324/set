package set_test

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/min1324/set"
)

type mapOp string

const (
	opLoad          = mapOp("Load")
	opLoadOrStore   = mapOp("LoadOrStore")
	opLoadAndDelete = mapOp("LoadAndDelete")
)

var mapOps = [...]mapOp{opLoad, opLoadOrStore, opLoadAndDelete}

// mapCall is a quick.Generator for calls on mapInterface.
type mapCall struct {
	op mapOp
	k  uint32
}

func (c mapCall) apply(m setInterface) (uint32, bool) {
	switch c.op {
	case opLoad:
		return c.k, m.Load(c.k)
	case opLoadOrStore:
		return c.k, m.LoadOrStore(c.k)
	case opLoadAndDelete:
		return c.k, m.LoadAndDelete(c.k)
	default:
		panic("invalid mapOp")
	}
}

type mapResult struct {
	value uint32
	ok    bool
}

func randValue(r *rand.Rand) uint32 {
	return uint32(rand.Int31n(32 * 5))
}

func (mapCall) Generate(r *rand.Rand, size int) reflect.Value {
	c := mapCall{op: mapOps[rand.Intn(len(mapOps))], k: randValue(r)}
	return reflect.ValueOf(c)
}

func applyCalls(m setInterface, calls []mapCall) (results []mapResult, final map[interface{}]interface{}) {
	for _, c := range calls {
		v, ok := c.apply(m)
		results = append(results, mapResult{v, ok})
	}

	final = make(map[interface{}]interface{})
	m.Range(func(x uint32) bool {
		final[x] = true
		return true
	})

	return results, final
}

func applySet(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	return applyCalls(new(set.IntSet), calls)
}

func applyMutex(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
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
		t.Fatalf("after adds Null not false")
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
}
