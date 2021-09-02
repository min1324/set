package set_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/min1324/set"
)

type opTyp int

const (
	opTypUnion opTyp = iota
	opTypInter
	opTypDiffe
	opTypComen
)

type args struct {
	name string
	val  set.Set
}

type opArgsRange struct {
	cap int
	x   int
	y   int
}

type opArgs struct {
	arg1 opArgsRange
	arg2 opArgsRange
	op   opTyp
}

type opTestFunc func(r opArgs, args1, args2 args)
type opFunc func(s, t set.Set) set.Set

func miss(want, got set.Set) (miss []uint32) {
	w := set.Items(want)
	g := set.Items(got)
	miss = make([]uint32, 0, initSize)
	i, j := 0, 0
	for i < len(w) && j < len(g) {
		if w[i] != g[j] {
			miss = append(miss, w[i])
			i += 1
		}
		i += 1
		j += 1
	}
	for ; i < len(w); i++ {
		miss = append(miss, w[i])
	}
	return miss
}

func (o opArgs) getOpFunc() opFunc {
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

func (o opArgs) getWant() (x, y set.Set) {
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

func opArgsTest(t *testing.T, r opArgs, f opTestFunc) {

	var (
		cap1 = r.arg1.cap
		cap2 = r.arg1.cap

		// keep: x1<x2<y1<y2
		x1 = r.arg1.x
		x2 = r.arg2.x
		y1 = r.arg1.y
		y2 = r.arg2.y
	)

	arg1 := []args{
		{
			name: "getStatic",
			val:  getStatic(cap1, x1, y1),
		},
		{
			name: "getTrends",
			val:  getDynamic(cap1, x1, y1),
		},

		{
			name: "getMutexSet",
			val:  getMutexSet(cap1, x1, y1),
		},
	}
	arg2 := []args{
		{
			name: "getStatic",
			val:  getStatic(cap2, x2, y2),
		},
		{
			name: "getTrends",
			val:  getDynamic(cap2, x2, y2),
		},
		{
			name: "getMutexSet",
			val:  getMutexSet(cap2, x2, y2),
		},
	}
	for _, x := range arg1 {
		for _, y := range arg2 {
			t.Run(x.name+"+"+y.name, func(t *testing.T) {
				f(r, x, y)
			})
		}
	}
}

func opMap(t *testing.T, r opArgs) {
	opArgsTest(t, r, func(r opArgs, x, y args) {
		wantxy, wantyx := r.getWant()
		SetFunc := r.getOpFunc()
		if got := SetFunc(x.val, y.val); !set.Equal(got, wantxy) {
			miss := miss(wantxy, got)
			t.Errorf("%v miss:%v ", x.name+"+"+y.name, fmt.Sprintf("%v ", miss))
		}
		if got := SetFunc(y.val, x.val); !set.Equal(got, wantyx) {
			miss := miss(wantyx, got)
			t.Errorf("miss:%v ", fmt.Sprintf("%v ", miss))
		}
	})
}

func TestUnion(t *testing.T) {
	opMap(t, opArgs{
		arg1: opArgsRange{
			cap: 100,
			x:   0,
			y:   50,
		},
		arg2: opArgsRange{
			cap: 100,
			x:   20,
			y:   80,
		},
		op: opTypUnion,
	})
}

func TestIntersect(t *testing.T) {
	opMap(t, opArgs{
		arg1: opArgsRange{
			cap: 100,
			x:   0,
			y:   50,
		},
		arg2: opArgsRange{
			cap: 100,
			x:   20,
			y:   80,
		},
		op: opTypInter,
	})
}

func TestDifference(t *testing.T) {
	opMap(t, opArgs{
		arg1: opArgsRange{
			cap: 100,
			x:   000,
			y:   50,
		},
		arg2: opArgsRange{
			cap: 100,
			x:   20,
			y:   80,
		},
		op: opTypDiffe,
	})
}

func TestComplement(t *testing.T) {
	opMap(t, opArgs{
		arg1: opArgsRange{
			cap: 100,
			x:   0,
			y:   50,
		},
		arg2: opArgsRange{
			cap: 100,
			x:   20,
			y:   80,
		},
		op: opTypComen,
	})
}

func TestEqual(t *testing.T) {
	m := getStatic(100, 0, 50)
	n := getStatic(100, 0, 40)
	if set.Equal(m, n) {
		t.Errorf("want:m!=n,got true")
	}
	m = getStatic(100, 0, 50)
	n = getStatic(80, 0, 50)
	if !set.Equal(m, n) {
		t.Errorf("want:m=n,got false")
	}

	tm := getDynamic(0, 0, 50)
	tn := getDynamic(80, 0, 50)
	if !set.Equal(tm, tn) {
		t.Errorf("want:tm=tn,got false")
	}

	tm = getDynamic(40, 0, 50)
	tn = getDynamic(80, 0, 50)
	if set.Equal(tm, tn) {
		t.Errorf("want:tm!=tn,got true")
	}
	// all type equal
	opArgsTest(t, opArgs{
		arg1: opArgsRange{
			cap: 100,
			x:   0,
			y:   50,
		},
		arg2: opArgsRange{
			cap: 100,
			x:   0,
			y:   50,
		},
	}, func(r opArgs, x, y args) {
		name := x.name + "+" + y.name
		if !set.Equal(x.val, y.val) {
			miss := miss(x.val, y.val)
			t.Errorf("%s miss:%v ", name, fmt.Sprintf("%v ", miss))
		}
	})
}

func TestCopy(t *testing.T) {
	cap := 100
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
				s: set.NewDynamic(cap, arg...),
			},
			want: set.NewDynamic(cap, arg...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Copy(tt.args.s); !set.Equal(got, tt.want) {
				miss := miss(tt.want, got)
				t.Errorf("%s miss:%v ", tt.name, fmt.Sprintf("%v ", miss))
			}
		})
	}
}

func TestToStatic(t *testing.T) {
	cap := 100
	var s set.Dynamic
	s.OnceInit(cap)
	for i := 0; i < cap; i++ {
		s.Store(uint32(i))
	}
	r := set.Copy(&s)
	type args struct {
		s *set.Dynamic
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
	cap := 1000
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
		want *set.Dynamic
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

func TestItems(t *testing.T) {
	type args struct {
		s set.Set
	}
	tests := []struct {
		name string
		args args
		want []uint32
	}{
		// TODO: Add test cases.

		{
			name: "getStatic",
			args: args{
				s: getStatic(10, 0, 10),
			},
			want: []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name: "getTrends",
			args: args{
				s: getDynamic(10, 0, 10),
			},
			want: []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Items(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Items() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNull(t *testing.T) {
	type args struct {
		s set.Set
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "null",
			args: args{
				s: getStatic(100, 0, 0),
			},
			want: true,
		},
		{
			name: "not null",
			args: args{
				s: getStatic(100, 0, 10),
			},
			want: false,
		},
		{
			name: "null",
			args: args{
				s: getDynamic(100, 0, 0),
			},
			want: true,
		},
		{
			name: "not null",
			args: args{
				s: getDynamic(100, 0, 10),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Null(tt.args.s); got != tt.want {
				t.Errorf("Null() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSize(t *testing.T) {
	type args struct {
		s set.Set
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
		{
			name: "getStatic",
			args: args{
				s: getStatic(100, 5, 10),
			},
			want: 5,
		},
		{
			name: "getStatic null",
			args: args{
				s: getStatic(100, 0, 0),
			},
			want: 0,
		},
		{
			name: "getTrends",
			args: args{
				s: getDynamic(100, 5, 10),
			},
			want: 5,
		},
		{
			name: "getTrends",
			args: args{
				s: getDynamic(100, 5, 5),
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Size(tt.args.s); got != tt.want {
				t.Errorf("Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClear(t *testing.T) {
	type args struct {
		s set.Set
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{
			name: "getStatic",
			args: args{
				s: getStatic(100, 10, 20),
			},
		},
		{
			name: "getTrends",
			args: args{
				s: getDynamic(100, 10, 20),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set.Clear(tt.args.s)
			tt.args.s.Range(func(x uint32) bool {
				t.Errorf("clear has: %v", x)
				return true
			})
		})
	}
}

func maxmin(x, y int) (max, min int) {
	if x > y {
		return x, y
	}
	return y, x
}
