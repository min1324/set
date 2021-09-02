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
	m.OnceInit(int(maximum))
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

type applyFunc func(calls []setCall) ([]setResult, map[interface{}]interface{})

func applyStatic(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(set.Static), calls)
}

func applyTrends(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(set.Static), calls)
}

func applyMutex(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(new(MutexSet), calls)
}

// func applyOpt32(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewOption32(int(maximum)), calls)
// }

// func applyOpt31(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewOption31(int(maximum)), calls)
// }

// func applyOpt16(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewOption16(int(maximum)), calls)
// }

// func applyOpt15(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewOption15(int(maximum)), calls)
// }

// func applyBase(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewBase(int(maximum)), calls)
// }

// func applyFasten(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewDynamic(int(maximum)), calls)
// }

func applyMap(t *testing.T, standard applyFunc) {
	for _, m := range [...]applyFunc{
		applyStatic,
		// applyTrends,
		// applyOpt15,
		// applyOpt16,
		// applyOpt31,
		// applyOpt32,
	} {
		name := fmt.Sprintf("%T", standard) + "+" + fmt.Sprintf("%T", m)
		t.Run(name, func(t *testing.T) {
			if err := quick.CheckEqual(standard, m, nil); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAll(t *testing.T) {
	for _, m := range [...]applyFunc{
		applyStatic,
		// applyTrends,
		// applyOpt15,
		// applyOpt16,
		// applyOpt31,
		// applyOpt32,
	} {
		applyMap(t, m)
	}
}

func TestMutex(t *testing.T) {
	applyMap(t, applyMutex)
}

func TestStatic(t *testing.T) {
	applyMap(t, applyStatic)
}

func TestTrends(t *testing.T) {
	applyMap(t, applyTrends)
}

// func TestOpt32(t *testing.T) {
// 	applyMap(t, applyOpt32)
// }

// func TestBase(t *testing.T) {
// 	applyMap(t, applyBase)
// }

// func TestFasten(t *testing.T) {
// 	applyMap(t, applyFasten)
// }

// func TestOpt31(t *testing.T) {
// 	applyMap(t, applyOpt31)
// }

// func TestOpt16(t *testing.T) {
// 	applyMap(t, applyOpt16)
// }

// func TestOpt15(t *testing.T) {
// 	applyMap(t, applyOpt15)
// }

func TestTrendsGrow(t *testing.T) {
	src := getTrends(0, 0, 5000)
	dst := getTrends(5000, 0, 5000)
	if !set.Equal(src, dst) {
		t.Errorf("not grow:%v", src)
	}
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
		&set.Static{},
		// &set.Trends{},
		// &MutexSet{},
		// set.NewBase(100, 5),
		// &set.Dynamic{},
		// set.NewIntOpt(preInitSize),
		// set.NewVarOpt(preInitSize),
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

var raceOps = [...]setOp{
	opLoad,
	opStore,
	opDelete,
	opLoadOrStore,
	opLoadAndDelete,
	opRange,
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
