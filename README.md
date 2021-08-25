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

### 交集

Intersect 交集 `P` 属于 `S` 或属于 `T` 的元素为元素的集合: `P = S∩T` 。

### 并集

Union 并集 `P` 属于 `S` 并且属于 `T` 的元素为元素的集合: `P = S∪T` 。

### 差集

Difference 差集 `P` 属于 `S` 并且不属于 `T` 的元素为元素的集合: `P = S-T` 。

### 补集

Complement 补集 `P` 属于 `S` 并且不属于 `T` 和不属于 `S` 并且属于 `T` 的元素为元素的集合: `P = S∩T' ∪ S'∩T` 。