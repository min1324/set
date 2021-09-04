package set

import "sort"

// 实用于范围小，元素重复次数多的场景
// 空间复杂度：2倍不重复元素个数
// 时间复杂度：O(3N)
func Sort(array []int32) {
	if len(array) < 1 {
		return
	}
	m := make(map[int]int)
	var s Base

	// 获取范围
	var min, max int32
	for i := range array {
		if min > array[i] {
			min = array[i]
		}
		if max < array[i] {
			max = array[i]
		}
	}
	srange := max - min
	if int(srange) > 2*len(array) {
		sort.Slice(array, func(i, j int) bool { return array[i] < array[j] })
		return
	}
	s.Init(int(max), int(min))
	// 排序
	for i := range array {
		// 记录顺序
		s.Add(array[i])

		// 记录次数
		num := m[int(array[i])]
		m[int(array[i])] = num + 1
	}
	// 统计结果
	j := 0
	s.Range(func(x int32) bool {
		num := m[int(x)]
		for i := 0; i < num; i++ {
			array[j] = x
			j += 1
		}
		return true
	})
}
