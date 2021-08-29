package set_test

import (
	"fmt"

	"github.com/min1324/set"
)

func ExampleSliceSet_initSet() {
	s := new(set.SliceSet)
	s.Adds(10000)
	s.Range(func(x uint32) bool {
		fmt.Printf("%d ", x)
		return true
	})
	// Output:
	// 10000
}

func ExampleInit() {
	var s set.SliceSet
	load, ok := s.LoadOrStore(0)
	if !ok {
		fmt.Println("!ok")
	}
	if load {
		fmt.Println("load")
	}
	fmt.Println(s.Load(0))
	fmt.Println(s.String())
	// Output:
	// true
	// {0}
}

func ExampleSliceSet_range() {
	s := set.NewSlice(100, 0, 1, 2, 31, 32, 63, 64, 91, 92, 99, 100, 101, 131)
	s.Store(7)
	s.Delete(2)
	s.Range(func(x uint32) bool {
		fmt.Printf("%d ", x)
		return true
	})
	// Output:
	// 0 1 7 31 32 63 64 91 92 99 100
}

func ExampleUnion() {
	s := set.NewSlice(36, 0, 1, 2, 3, 4, 5)
	p := set.NewSlice(100, 4, 5, 6, 7, 8)
	u := set.Union(s, p)
	fmt.Println(u)

	m := set.New(100, 4, 5, 6, 7, 8)
	n := set.Union(s, m)
	fmt.Println(n)
	// Output:
	// {0 1 2 3 4 5 6 7 8}
	// {0 1 2 3 4 5 6 7 8}
}

func ExampleIntersect() {
	s := set.NewSlice(36, 0, 1, 2, 3, 4, 5)
	p := set.NewSlice(100, 4, 5, 6, 7, 8)
	u := set.Intersect(s, p)
	fmt.Println(u)

	m := set.New(100, 4, 5, 6, 7, 8)
	n := set.Intersect(s, m)
	fmt.Println(n)
	// Output:
	// {4 5}
	// {4 5}
}

func ExampleDifference() {
	s := set.NewSlice(36, 0, 1, 2, 3, 4, 5)
	p := set.NewSlice(100, 4, 5, 6, 7, 8)
	u := set.Difference(s, p)
	fmt.Println(u)

	m := set.NewSlice(100, 4, 5, 6, 7, 8)
	n := set.Difference(s, m)
	fmt.Println(n)
	// Output:
	// {0 1 2 3}
	// {0 1 2 3}
}

func ExampleComplement() {
	s := set.NewSlice(36, 0, 1, 2, 3, 4, 5)
	p := set.NewSlice(100, 4, 5, 6, 7, 8)
	u := set.Complement(s, p)
	fmt.Println(u)

	m := set.New(100, 4, 5, 6, 7, 8)
	n := set.Complement(s, m)
	fmt.Println(n)
	// Output:
	// {0 1 2 3 6 7 8}
	// {0 1 2 3 6 7 8}
}
