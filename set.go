package set

// Equal return set if equal, s <==> t
// worst time complexity: O(N)
// best  time complexity: O(1)
func Equal(s, t *SliceSet) bool {
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	minLen := min(sLen, tLen)
	for i := 0; i < minLen; i++ {
		if s.load(i) != t.load(i) {
			return false
		}
	}
	if sLen == tLen {
		return true
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			if s.load(i) != 0 {
				return false
			}
		}
	} else {
		for i := minLen; i < tLen; i++ {
			if t.load(i) != 0 {
				return false
			}
		}
	}
	return true
}

// Union return the union set of s and t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Union(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.init(max(s.Cap(), t.Cap()))

	// [0-minLen]
	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)|t.load(i))
	}
	// [minLen-maxLen]
	if sLen < tLen {
		for i := minLen; i < maxLen; i++ {
			p.store(i, t.load(i))
		}
	} else {
		for i := minLen; i < maxLen; i++ {
			p.store(i, s.load(i))
		}
	}

	return &p
}

// Intersect return the intersection set of s and t
// item in s and t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Intersect(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	minLen := min(sLen, tLen)
	p.init(min(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&t.load(i))
	}
	return &p
}

// Difference return the difference set of s and t
// item in s and not in t
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Difference(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	minLen := min(sLen, tLen)
	p.init(s.Cap())

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)&^t.load(i))
	}
	if sLen > tLen {
		for i := minLen; i < sLen; i++ {
			p.store(i, s.load(i))
		}
	}

	return &p
}

// Complement return the complement set of s and t
// item in s but not in t, and not in s but in t.
// worst time complexity: O(N)
// best  time complexity: O(N/32)
func Complement(s, t *SliceSet) *SliceSet {
	var p SliceSet
	sn, tn := s.getNode(), t.getNode()
	sLen, tLen := int(sn.getLen()), int(tn.getLen())
	maxLen, minLen := maxmin(sLen, tLen)
	p.init(max(s.Cap(), t.Cap()))

	for i := 0; i < minLen; i++ {
		p.store(i, s.load(i)^t.load(i))
	}
	for i := minLen; i < maxLen; i++ {
		if sLen > tLen {
			p.store(i, s.load(i))
		} else {
			p.store(i, t.load(i))
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
