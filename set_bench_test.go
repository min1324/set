package set_test

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/min1324/set"
)

const (
	preInitSize = 1 << 25
)

type bench struct {
	setup func(*testing.B, Interface)
	perG  func(b *testing.B, pb *testing.PB, i int, m Interface)
}

func benchMap(b *testing.B, bench bench) {
	for _, m := range [...]Interface{
		&set.Dynamic{},
		&set.Static{},
		// set.NewOption15(preInitSize),
		// set.NewOption16(preInitSize),
		// set.NewOption31(preInitSize),
		// set.NewOption32(preInitSize),
		// &set.Base{},
		// &set.Dynamic{},
		// &MutexSet{},
	} {
		b.Run(fmt.Sprintf("%T", m), func(b *testing.B) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(Interface)
			// setup
			if bench.setup != nil {
				bench.setup(b, m)
			}
			m.OnceInit(preInitSize)
			b.ResetTimer()
			var i int64
			b.RunParallel(func(pb *testing.PB) {
				id := int(atomic.AddInt64(&i, 1) - 1)
				bench.perG(b, pb, (id * b.N), m)
			})
		})
	}
}

func BenchmarkLoadMostlyHits(b *testing.B) {
	const hits, misses = 1023, 1

	benchMap(b, bench{
		setup: func(_ *testing.B, m Interface) {
			m.OnceInit(1 << 10)
			for i := 0; i < hits; i++ {
				m.LoadOrStore(uint32(i))
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Load(uint32(i % (hits + misses)))
			}
		},
	})
}

func BenchmarkLoadMostlyMisses(b *testing.B) {
	const hits, misses = 1, 1023

	benchMap(b, bench{
		setup: func(_ *testing.B, m Interface) {
			m.OnceInit(1 << 10)
			for i := 0; i < hits; i++ {
				m.LoadOrStore(uint32(i))
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Load(uint32(i % (hits + misses)))
			}
		},
	})
}

func BenchmarkLoadOrStoreBalanced(b *testing.B) {
	const hits, misses = 128, 128

	benchMap(b, bench{
		setup: func(b *testing.B, m Interface) {
			m.OnceInit(1 << 10)
			for i := 0; i < hits; i++ {
				m.LoadOrStore(uint32(i))
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				j := i % (hits + misses)
				if j < hits {
					if loaded, _ := m.LoadOrStore(uint32(j)); !loaded {
						b.Fatalf("unexpected miss for %v", j)
					}
				} else {
					if loaded, _ := m.LoadOrStore(uint32(i)); loaded {
						b.Fatalf("failed to store %v: existing value %v", i, i)
					}
				}
			}
		},
	})
}

func BenchmarkLoadOrStoreUnique(b *testing.B) {
	benchMap(b, bench{
		setup: func(b *testing.B, m Interface) {
			m.OnceInit(1 << 25)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.LoadOrStore(uint32(i))
			}
		},
	})
}

func BenchmarkLoadOrStoreCollision(b *testing.B) {
	benchMap(b, bench{
		setup: func(_ *testing.B, m Interface) {
			m.OnceInit(1 << 10)
			m.LoadOrStore(uint32(0))
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.LoadOrStore(uint32(0))
			}
		},
	})
}

func BenchmarkLoadAndDeleteBalanced(b *testing.B) {
	const hits, misses = 128, 128

	benchMap(b, bench{
		setup: func(b *testing.B, m Interface) {
			m.OnceInit(1 << 10)
			for i := 0; i < hits; i++ {
				m.LoadOrStore(uint32(i))
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				j := i % (hits + misses)
				if j < hits {
					m.LoadAndDelete(uint32(j))
				} else {
					m.LoadAndDelete(uint32(i))
				}
			}
		},
	})
}

func BenchmarkLoadAndDeleteUnique(b *testing.B) {
	benchMap(b, bench{
		setup: func(b *testing.B, m Interface) {
			m.OnceInit(preInitSize)
			for i := 0; i < preInitSize; i++ {
				m.Store(uint32(i))
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.LoadAndDelete(uint32(i))
			}
		},
	})
}

func BenchmarkLoadAndDeleteCollision(b *testing.B) {
	benchMap(b, bench{
		setup: func(_ *testing.B, m Interface) {
			m.OnceInit(1 << 10)
			m.LoadOrStore(0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.LoadAndDelete(0)
			}
		},
	})
}

func BenchmarkRange(b *testing.B) {
	const mapSize = 1 << 10

	benchMap(b, bench{
		setup: func(b *testing.B, m Interface) {
			m.OnceInit(mapSize)
			for i := 0; i < mapSize; i++ {
				if !m.Store(uint32(i)) {
					b.Errorf("not store:%d", i)
				}
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Range(func(x uint32) bool { return true })
			}
		},
	})
}

type opType int

const (
	opUnion opType = iota
	opIntersect
	opDifference
	opComplement
)

type opBench struct {
	name string
	x    Interface
	y    Interface
}

func (op opBench) call(t opType, invert bool) {
	if invert {
		switch t {
		case opUnion:
			set.Union(op.y, op.x)
		case opIntersect:
			set.Intersect(op.y, op.x)
		case opDifference:
			set.Difference(op.y, op.x)
		case opComplement:
			set.Complement(op.y, op.x)
		}
	} else {
		switch t {
		case opUnion:
			set.Union(op.x, op.y)
		case opIntersect:
			set.Intersect(op.x, op.y)
		case opDifference:
			set.Difference(op.x, op.y)
		case opComplement:
			set.Complement(op.x, op.y)
		}
	}
}

func getStatic(cap, m, n int) *set.Static {
	var s set.Static
	s.OnceInit(cap)
	for i := m; i < n; i++ {
		s.Store(uint32(i))
	}
	return &s
}

func getTrends(cap, m, n int) *set.Dynamic {
	var s set.Dynamic
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

// func getOpt15(cap, m, n int) *set.Option {
// 	s := set.NewOption15(cap)
// 	s.OnceInit(cap)
// 	for i := m; i < n; i++ {
// 		s.Store(uint32(i))
// 	}
// 	return s
// }

// func getOpt16(cap, m, n int) *set.Option {
// 	s := set.NewOption16(cap)
// 	s.OnceInit(cap)
// 	for i := m; i < n; i++ {
// 		s.Store(uint32(i))
// 	}
// 	return s
// }

// func getOpt31(cap, m, n int) *set.Option {
// 	s := set.NewOption31(cap)
// 	s.OnceInit(cap)
// 	for i := m; i < n; i++ {
// 		s.Store(uint32(i))
// 	}
// 	return s
// }

// func getOpt32(cap, m, n int) *set.Option {
// 	s := set.NewOption32(cap)
// 	s.OnceInit(cap)
// 	for i := m; i < n; i++ {
// 		s.Store(uint32(i))
// 	}
// 	return s
// }

func call(b *testing.B, op opType, invert bool) {
	const (
		cap1   = 200
		start1 = 0
		end1   = 100

		cap2   = 200
		start2 = 40
		end2   = 150
	)

	for _, v := range [...]opBench{
		{"SS", getStatic(cap1, start1, end1), getStatic(cap2, start2, end2)},
		{"ST", getStatic(cap1, start1, end1), getTrends(cap2, start2, end2)},
		// {"S15", getStatic(cap1, start1, end1), getOpt15(cap2, start2, end2)},
		// {"S16", getStatic(cap1, start1, end1), getOpt16(cap2, start2, end2)},
		// {"S31", getStatic(cap1, start1, end1), getOpt31(cap2, start2, end2)},
		// {"S32", getStatic(cap1, start1, end1), getOpt32(cap2, start2, end2)},

		{"TT", getTrends(cap1, start1, end1), getTrends(cap2, start2, end2)},
		// {"T15", getTrends(cap1, start1, end1), getOpt15(cap2, start2, end2)},
		// {"T16", getTrends(cap1, start1, end1), getOpt16(cap2, start2, end2)},
		// {"T31", getTrends(cap1, start1, end1), getOpt31(cap2, start2, end2)},
		// {"T32", getTrends(cap1, start1, end1), getOpt32(cap2, start2, end2)},

		// {"1515", getOpt15(cap1, start1, end1), getOpt15(cap2, start2, end2)},
		// {"1516", getOpt15(cap1, start1, end1), getOpt16(cap2, start2, end2)},
		// {"1531", getOpt15(cap1, start1, end1), getOpt31(cap2, start2, end2)},
		// {"1532", getOpt15(cap1, start1, end1), getOpt32(cap2, start2, end2)},

		// {"1616", getOpt16(cap1, start1, end1), getOpt16(cap2, start2, end2)},
		// {"1631", getOpt16(cap1, start1, end1), getOpt31(cap2, start2, end2)},
		// {"1632", getOpt16(cap1, start1, end1), getOpt32(cap2, start2, end2)},

		// {"3131", getOpt31(cap1, start1, end1), getOpt31(cap2, start2, end2)},
		// {"3132", getOpt31(cap1, start1, end1), getOpt32(cap2, start2, end2)},

		// {"3232", getOpt32(cap1, start1, end1), getOpt32(cap2, start2, end2)},

		{"MM", getMutexSet(cap1, start1, end1), getMutexSet(cap2, start2, end2)},
		{"MS", getMutexSet(cap1, start1, end1), getStatic(cap2, start2, end2)},
		{"MT", getMutexSet(cap1, start1, end1), getTrends(cap2, start2, end2)},
		// {"M15", getMutexSet(cap1, start1, end1), getOpt15(cap2, start2, end2)},
		// {"M16", getMutexSet(cap1, start1, end1), getOpt16(cap2, start2, end2)},
		// {"M31", getMutexSet(cap1, start1, end1), getOpt31(cap2, start2, end2)},
		// {"M32", getMutexSet(cap1, start1, end1), getOpt32(cap2, start2, end2)},
	} {
		b.Run(v.name, func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					v.call(op, invert)
				}
			})
		})
	}
}

func BenchmarkUnion(b *testing.B) {
	call(b, opUnion, false)
}

func BenchmarkIntersect(b *testing.B) {
	call(b, opIntersect, false)
}

func BenchmarkDifference(b *testing.B) {
	call(b, opDifference, false)
}

func BenchmarkComplement(b *testing.B) {
	call(b, opComplement, false)
}
