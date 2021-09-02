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

func applyOpt32(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewOption32(int(maximum)), calls)
}

func applyOpt31(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewOption31(int(maximum)), calls)
}

func applyOpt16(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewOption16(int(maximum)), calls)
}

func applyOpt15(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewOption15(int(maximum)), calls)
}

// func applyBase(calls []setCall) ([]setResult, map[interface{}]interface{}) {
// 	return applyCalls(set.NewBase(int(maximum)), calls)
// }

func applyFasten(calls []setCall) ([]setResult, map[interface{}]interface{}) {
	return applyCalls(set.NewFasten(int(maximum)), calls)
}

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

func TestOpt32(t *testing.T) {
	applyMap(t, applyOpt32)
}

// func TestBase(t *testing.T) {
// 	applyMap(t, applyBase)
// }

func TestFasten(t *testing.T) {
	applyMap(t, applyFasten)
}

func TestOpt31(t *testing.T) {
	applyMap(t, applyOpt31)
}

func TestOpt16(t *testing.T) {
	applyMap(t, applyOpt16)
}

func TestOpt15(t *testing.T) {
	applyMap(t, applyOpt15)
}

func getStatic(cap, m, n int) *set.Static {
	var s set.Static
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

func getTrends(cap, m, n int) *set.Trends {
	var s set.Trends
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

func getOpt15(cap, m, n int) *set.Option {
	s := set.NewOption15(cap)
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return s
}

func getOpt16(cap, m, n int) *set.Option {
	s := set.NewOption16(cap)
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return s
}

func getOpt31(cap, m, n int) *set.Option {
	s := set.NewOption31(cap)
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return s
}

func getOpt32(cap, m, n int) *set.Option {
	s := set.NewOption32(cap)
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return s
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
		// &set.Static{},
		// &set.Trends{},
		// &MutexSet{},
		// set.NewBase(100, 5),
		&set.Fasten{},
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

func TestEqual(t *testing.T) {
	iq := getStatic(initCap, initM, initN)
	sq := getStatic(initCap, initM, initN)
	mq := getStatic(initCap, initM, initN)
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

type opTyp int

const (
	opTypUnion opTyp = iota
	opTypInter
	opTypDiffe
	opTypComen
)

type opArgs struct {
	cap int
	x   int
	y   int
}

type opResualt struct {
	arg1 opArgs
	arg2 opArgs
	op   opTyp
}

type opfunc func(s, t set.Set) set.Set

func (o opResualt) getFunc() opfunc {
	switch o.op {
	case opTypUnion:
		return set.Union
	case opTypInter:
		return set.Intersect
	case opTypDiffe:
		return set.Difference
	case opTypComen:
		return set.Complement
	}
	return nil
}

func (o opResualt) getWant() (x, y set.Set) {
	// keep: x1<x2<y1<y2
	switch o.op {
	case opTypUnion:
		max, _ := maxmin(o.arg1.cap, o.arg2.cap)
		x = getStatic(max, o.arg1.x, o.arg2.y)
		y = x
		return x, y
	case opTypInter:
		max, _ := maxmin(o.arg1.cap, o.arg2.cap)
		x = getStatic(max, o.arg2.x, o.arg1.y)
		y = x
		return x, y
	case opTypDiffe:
		x = getStatic(o.arg1.cap, o.arg1.x, o.arg2.x)
		y = getStatic(o.arg2.cap, o.arg1.y, o.arg2.y)
		return x, y
	case opTypComen:
		max, _ := maxmin(o.arg1.cap, o.arg2.cap)
		x = set.Union(getStatic(max, o.arg1.x, o.arg2.x),
			getStatic(max, o.arg1.y, o.arg2.y))
		y = x
		return x, y
	}
	return nil, nil
}

func opMap(t *testing.T, r opResualt) {
	type args struct {
		name string
		val  set.Set
	}
	var (
		cap1 = r.arg1.cap
		cap2 = r.arg1.cap

		// keep: x1<x2<y1<y2
		x1 = r.arg1.x
		x2 = r.arg2.x
		y1 = r.arg1.y
		y2 = r.arg2.y
	)
	wantxy, wantyx := r.getWant()
	SetFunc := r.getFunc()

	arg1 := []args{
		{
			name: "getStatic",
			val:  getStatic(cap1, x1, y1),
		},
		{
			name: "getTrends",
			val:  getTrends(cap1, x1, y1),
		},
		{
			name: "getMutexSet",
			val:  getMutexSet(cap1, x1, y1),
		},
		{
			name: "getOpt15",
			val:  getOpt15(cap1, x1, y1),
		},
		{
			name: "getOpt16",
			val:  getOpt16(cap1, x1, y1),
		},
		{
			name: "getOpt31",
			val:  getOpt31(cap1, x1, y1),
		},
		{
			name: "getOpt32",
			val:  getOpt32(cap1, x1, y1),
		},
	}
	arg2 := []args{
		{
			name: "getStatic",
			val:  getStatic(cap2, x2, y2),
		},
		{
			name: "getTrends",
			val:  getTrends(cap2, x2, y2),
		},
		{
			name: "getMutexSet",
			val:  getMutexSet(cap2, x2, y2),
		},
		{
			name: "getOpt15",
			val:  getOpt15(cap2, x2, y2),
		},
		{
			name: "getOpt16",
			val:  getOpt16(cap2, x2, y2),
		},
		{
			name: "getOpt31",
			val:  getOpt31(cap2, x2, y2),
		},
		{
			name: "getOpt32",
			val:  getOpt32(cap2, x2, y2),
		},
	}
	for _, x := range arg1 {
		for _, y := range arg2 {
			t.Run(x.name+"+"+y.name, func(t *testing.T) {
				if got := SetFunc(x.val, y.val); !set.Equal(got, wantxy) {
					wantItem := set.Items(wantxy)
					gotItem := set.Items(got)
					i, j := 0, 0
					for i < len(wantItem) && j < len(gotItem) {
						if wantItem[i] != gotItem[j] {
							t.Errorf("miss:%d ", wantItem[i])
							i += 1
						}
						i += 1
						j += 1
					}
					for ; i < len(wantItem); i++ {
						t.Errorf("miss:%d ", wantItem[i])
					}
				}
				if got := SetFunc(y.val, x.val); !set.Equal(got, wantyx) {
					wantItem := set.Items(wantyx)
					gotItem := set.Items(got)
					i, j := 0, 0
					for i < len(wantItem) && j < len(gotItem) {
						if wantItem[i] != gotItem[j] {
							t.Errorf("miss:%d ", wantItem[i])
							i += 1
						}
						i += 1
						j += 1
					}
					for ; i < len(wantItem); i++ {
						t.Errorf("miss:%d ", wantItem[i])
					}
				}
			})
		}
	}
}

func TestUnion(t *testing.T) {
	opMap(t, opResualt{
		arg1: opArgs{
			cap: 1000,
			x:   0,
			y:   500,
		},
		arg2: opArgs{
			cap: 1000,
			x:   200,
			y:   800,
		},
		op: opTypUnion,
	})
}

func TestIntersect(t *testing.T) {
	opMap(t, opResualt{
		arg1: opArgs{
			cap: 10000,
			x:   0,
			y:   5000,
		},
		arg2: opArgs{
			cap: 10000,
			x:   2000,
			y:   8000,
		},
		op: opTypInter,
	})
}

func TestDifference(t *testing.T) {
	opMap(t, opResualt{
		arg1: opArgs{
			cap: 10000,
			x:   000,
			y:   5000,
		},
		arg2: opArgs{
			cap: 10000,
			x:   2000,
			y:   8000,
		},
		op: opTypDiffe,
	})
}

func TestComplement(t *testing.T) {
	opMap(t, opResualt{
		arg1: opArgs{
			cap: 10000,
			x:   0,
			y:   5000,
		},
		arg2: opArgs{
			cap: 10000,
			x:   2000,
			y:   8000,
		},
		op: opTypComen,
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

func TestToStatic(t *testing.T) {
	cap := 1000
	var s set.Trends
	s.OnceInit(cap)
	for i := 0; i < cap; i++ {
		s.Store(uint32(i))
	}
	r := set.Copy(&s)
	type args struct {
		s *set.Trends
	}
	tests := []struct {
		name string
		args args
		want *set.Static
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
			if got := set.ToStatic(tt.args.s); !set.Equal(got, r) {
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

func TestToTrends(t *testing.T) {
	cap := 100000
	var s set.Static
	s.OnceInit(cap)
	for i := 0; i < cap; i++ {
		s.Store(uint32(i))
	}
	r := set.Copy(&s)
	type args struct {
		s *set.Static
	}
	tests := []struct {
		name string
		args args
		want *set.Trends
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
			if got := set.ToTrends(tt.args.s); !set.Equal(got, r) {
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

func TestCopy(t *testing.T) {
	cap := 10000
	arg := make([]uint32, cap)
	for i := 0; i < cap; i++ {
		arg[i] = uint32(i)
	}
	type args struct {
		s set.Set
	}
	tests := []struct {
		name string
		args args
		want set.Set
	}{
		// TODO: Add test cases.
		{
			name: "NewStatic",
			args: args{
				s: set.NewStatic(cap, arg...),
			},
			want: set.NewStatic(cap, arg...),
		},
		{
			name: "NewTrends",
			args: args{
				s: set.NewTrends(cap, arg...),
			},
			want: set.NewTrends(cap, arg...),
		},
		{
			name: "NewOpt16S",
			args: args{
				s: set.NewOption16(cap, arg...),
			},
			want: set.NewOption16(cap, arg...),
		},
		{
			name: "NewOpt16T",
			args: args{
				s: set.NewOption15(cap, arg...),
			},
			want: set.NewOption15(cap, arg...),
		},
		{
			name: "NewOpt31",
			args: args{
				s: set.NewOption31(cap, arg...),
			},
			want: set.NewOption31(cap, arg...),
		},
		{
			name: "NewOpt32",
			args: args{
				s: set.NewOption32(cap, arg...),
			},
			want: set.NewOption32(cap, arg...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Copy(tt.args.s); !set.Equal(got, tt.want) {
				t.Errorf("Copy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMsg(t *testing.T) {
	s := set.New(10, 0, 1, 2, 3, 4)
	a := set.New(30, 2, 3, 4, 5, 6, 7)
	type args struct {
		s set.Set
		t set.Set
	}
	tests := []struct {
		name     string
		args     args
		wantAs   []uint32
		wantAt   []uint32
		wantCmax int
		wantCmin int
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				s: s,
				t: a,
			},
			wantAs:   []uint32{0, 1, 2, 3, 4},
			wantAt:   []uint32{2, 3, 4, 5, 6, 7},
			wantCmax: 7,
			wantCmin: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAs, gotAt, gotCmax, gotCmin := getMsg(tt.args.s, tt.args.t)
			if !reflect.DeepEqual(gotAs, tt.wantAs) {
				t.Errorf("getMsg() gotAs = %v, want %v", gotAs, tt.wantAs)
			}
			if !reflect.DeepEqual(gotAt, tt.wantAt) {
				t.Errorf("getMsg() gotAt = %v, want %v", gotAt, tt.wantAt)
			}
			if gotCmax != tt.wantCmax {
				t.Errorf("getMsg() gotCmax = %v, want %v", gotCmax, tt.wantCmax)
			}
			if gotCmin != tt.wantCmin {
				t.Errorf("getMsg() gotCmin = %v, want %v", gotCmin, tt.wantCmin)
			}
		})
	}
}

func getMsg(s, t set.Set) (as, at []uint32, cmax, cmin int) {
	as = make([]uint32, 0, initSize)
	at = make([]uint32, 0, initSize)
	sm := uint32(0)
	s.Range(func(x uint32) bool {
		as = append(as, x)
		sm = x
		return true
	})
	tm := uint32(0)
	t.Range(func(x uint32) bool {
		at = append(at, x)
		tm = x
		return true
	})
	cmax, cmin = maxmin(int(sm), int(tm))
	return as, at, cmax, cmin
}

func maxmin(x, y int) (max, min int) {
	if x > y {
		return x, y
	}
	return y, x
}
