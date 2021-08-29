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
		&set.IntSet{},
		&set.SliceSet{},
		&MutexSet{},
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
			for i := 0; i < m.Cap(); i++ {
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
		setup: func(_ *testing.B, m Interface) {
			for i := 0; i < mapSize; i++ {
				m.Store(uint32(i))
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Range(func(x uint32) bool { return true })
			}
		},
	})
}
