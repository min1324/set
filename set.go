package set

// Equal return set if equal, s <==> t
// time complexity: O(N/32)
func Equal(s, t *IntSet) bool {
	sLen, tLen := len(s.items), len(t.items)
	if sLen != tLen {
		return false
	}
	for i := 0; i < sLen; i++ {
		if s.items[i] != t.items[i] {
			return false
		}
	}
	return true
}

// Union return the union set of s and t.
// time complexity: O(N/32)
func Union(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.items), len(t.items)
	maxLen, minLen := maxmin(sLen, tLen)
	p.items = make([]uint32, maxLen)

	// [0-minLen]
	for i := 0; i < minLen; i++ {
		p.items[i] = s.items[i] | t.items[i]
	}

	// [minLen-maxLen]
	if sLen < tLen {
		for i := minLen; i < maxLen; i++ {
			p.items[i] = t.items[i]
		}
	} else {
		for i := minLen; i < maxLen; i++ {
			p.items[i] = s.items[i]
		}
	}

	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
// time complexity: O(N/32)
func Intersect(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.items), len(t.items)
	minLen := min(sLen, tLen)
	p.items = make([]uint32, minLen)

	for i := 0; i < minLen; i++ {
		p.items[i] = s.items[i] & t.items[i]
	}

	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
// time complexity: O(N/32)
func Difference(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.items), len(t.items)
	minLen := min(sLen, tLen)
	p.items = make([]uint32, sLen)

	for i := 0; i < minLen; i++ {
		p.items[i] = s.items[i] &^ t.items[i]
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.items[i] = s.items[i]
		}
	}

	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// time complexity: O(N/32)
func Complement(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.items), len(t.items)
	maxLen, minLen := maxmin(sLen, tLen)
	p.items = make([]uint32, maxLen)

	for i := 0; i < minLen; i++ {
		p.items[i] = s.items[i] ^ t.items[i]
	}
	for i := minLen; i < maxLen; i++ {
		if sLen > tLen {
			p.items[i] = s.items[i]
		} else {
			p.items[i] = t.items[i]
		}
	}

	return &p
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxmin(x, y int) (max, min int) {
	if x > y {
		return x, y
	}
	return y, x
}
