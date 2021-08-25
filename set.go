package set

// Equal return set if equal, s <==> t
func Equal(s, t *IntSet) bool {
	sLen, tLen := len(s.dirty), len(t.dirty)
	if sLen != tLen {
		return false
	}
	for i := 0; i < sLen; i++ {
		if s.dirty[i] != t.dirty[i] {
			return false
		}
	}
	return true
}

// Union return the union set of s and t.
func Union(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.dirty), len(t.dirty)
	maxLen, minLen := maxmin(sLen, tLen)
	p.dirty = make([]uint32, maxLen)

	// [0-minLen]
	for i := 0; i < minLen; i++ {
		p.dirty[i] = s.dirty[i] | t.dirty[i]
	}

	// [minLen-maxLen]
	if sLen < tLen {
		for i := minLen; i < maxLen; i++ {
			p.dirty[i] = t.dirty[i]
		}
	} else {
		for i := minLen; i < maxLen; i++ {
			p.dirty[i] = s.dirty[i]
		}
	}

	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
func Intersect(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.dirty), len(t.dirty)
	minLen := min(sLen, tLen)
	p.dirty = make([]uint32, minLen)

	for i := 0; i < minLen; i++ {
		p.dirty[i] = s.dirty[i] & t.dirty[i]
	}

	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
func Difference(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.dirty), len(t.dirty)
	minLen := min(sLen, tLen)
	p.dirty = make([]uint32, sLen)

	for i := 0; i < minLen; i++ {
		p.dirty[i] = s.dirty[i] &^ t.dirty[i]
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.dirty[i] = s.dirty[i]
		}
	}

	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
func Complement(s, t *IntSet) *IntSet {
	var p IntSet
	sLen, tLen := len(s.dirty), len(t.dirty)
	maxLen, minLen := maxmin(sLen, tLen)
	p.dirty = make([]uint32, maxLen)

	for i := 0; i < minLen; i++ {
		p.dirty[i] = s.dirty[i] ^ t.dirty[i]
	}
	for i := minLen; i < maxLen; i++ {
		if sLen > tLen {
			p.dirty[i] = s.dirty[i]
		} else {
			p.dirty[i] = t.dirty[i]
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
