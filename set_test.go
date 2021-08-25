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
