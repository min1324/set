package set

// Equal return set if equal, s <==> t
// worst time complexity: O(N)
// best  time complexity: O(1)
func Equal(s, t *IntSet) bool {
	sLen, tLen := s.maxIndex(), t.maxIndex()
	minLen := min(sLen, tLen)
	for i := 0; i < minLen; i++ {
		if s.loadIdx(i) != t.loadIdx(i) {
			return false
		}
	}
	if sLen == tLen {
		return true
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			if s.loadIdx(i) != 0 {
				return false
			}
		}
	} else {
		for i := minLen; i < tLen; i++ {
			if t.loadIdx(i) != 0 {
				return false
			}
		}
	}
	return true
}

// Union return the union set of s and t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Union(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := s.maxIndex(), t.maxIndex()
	maxLen, minLen := maxmin(sLen, tLen)
	p.OnceInit(max(s.Cap(), t.Cap()))

	// [0-minLen]
	for i := 0; i < minLen; i++ {
		p.items[i] = s.loadIdx(i) | t.loadIdx(i)

	}
	// [minLen-maxLen]
	if sLen < tLen {
		for i := minLen; i < maxLen; i++ {
			p.items[i] = t.loadIdx(i)
		}
	} else {
		for i := minLen; i < maxLen; i++ {
			p.items[i] = s.loadIdx(i)
		}
	}

	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Intersect(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := s.maxIndex(), t.maxIndex()
	minLen := min(sLen, tLen)
	p.OnceInit(min(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.items[i] = s.loadIdx(i) & t.loadIdx(i)
	}

	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Difference(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := s.maxIndex(), t.maxIndex()
	minLen := min(sLen, tLen)
	p.OnceInit(s.Cap())

	for i := 0; i < minLen; i++ {
		p.items[i] = s.loadIdx(i) &^ t.loadIdx(i)
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.items[i] = s.loadIdx(i)
		}
	}

	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Complement(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := s.maxIndex(), t.maxIndex()
	maxLen, minLen := maxmin(sLen, tLen)
	p.OnceInit(max(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.items[i] = s.loadIdx(i) ^ t.loadIdx(i)
	}
	for i := minLen; i < maxLen; i++ {
		if sLen > tLen {
			p.items[i] = s.loadIdx(i)
		} else {
			p.items[i] = t.loadIdx(i)
		}
	}

	return &p
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
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
