# Set

[![Build Status](https://travis-ci.com/min1324/set.svg?branch=main)](https://travis-ci.com/min1324/set) [![codecov](https://codecov.io/gh/min1324/set/branch/main/graph/badge.svg)](https://codecov.io/gh/min1324/set) [![Go Report Card](https://goreportcard.com/badge/github.com/min1324/set)](https://goreportcard.com/report/github.com/min1324/set) [![GoDoc](https://godoc.org/github.com/min1324/set?status.png)](https://godoc.org/github.com/min1324/set) 

-----
## 定义

set是一个无序且不重复的元素集合。

## 位运算
| 操作符 |  名称  | 说明                           |
| :----: | :----: | :----------------------------- |
|  `&`   |  位与  | `AND` 运算，指定位清零的方式。 |
|  `|`   |  位或  | `OR` 运算，指定位置为 `1`。    |
|  `^`   |  异或  | `XOR` 运算，切换指定位上的值。 |
|  `&^`  | 位与非 | `AND NOT`运算，异或某位置。    |
| `<< `  |  左移  |                                |
|  `>>`  |  右移  |                                |

## 集合操作

设定有两个集合 `S` 和 `T` 。

1. Intersect 交集`P`: 属于`S`并且属于`T`的元素为元素的集合:`P = S∩T` 。
2. Union 并集`P`: 属于`S`或属于`T`的元素为元素的集合:`P = S∪T` 。
3. Difference 差集`P`: 属于`S`并且不属于`T`的元素为元素的集合:`P = S-T` 。
4. Complement 补集`P`: 属于`S`并且不属于`T`和不属于`S`并且属于`T`的元素为元素的集合:`P = (S∩T')∪(S'∩T)` 。

## Usage

###  Install

~~~bash
go get github.com/min1324/set
~~~

### Example

~~~go
package main

import (
	"fmt"
    
	"github.com/min1324/set"
)

func main() {
	var s, p set.IntSet
	s.Adds(1, 2, 3, 4, 5, 6)
	p.Adds(4, 5, 6, 7, 8, 9)
	fmt.Printf("S:%v\n", s.String())
	fmt.Printf("P:%v\n", p.String())

	fmt.Printf("Equal:%v\n", set.Equal(&s, &p))
	fmt.Printf("Union:%v\n", set.Union(&s, &p))
	fmt.Printf("Intersect:%v\n", set.Intersect(&s, &p))
	fmt.Printf("Difference:%v\n", set.Difference(&s, &p))
	fmt.Printf("Complement:%v\n", set.Complement(&s, &p))
}
// the result is:
S:{1 2 3 4 5 6}
P:{4 5 6 7 8 9}
Equal:false
Union:{1 2 3 4 5 6 7 8 9}
Intersect:{4 5 6}
Difference:{1 2 3}
Complement:{1 2 3 7 8 9}
~~~

