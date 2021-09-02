package set

const (
	// the max item can store in set.
	// total has slice: 1<<31, each slice can hold 31 item
	// so maximum = 1<<31*31
	// but the memory had not enough space.
	// set the maximum= 1<<24*31
	maximum uint32 = 1 << 24 * 31

	freezeBit = 1 << 31

	initCap  = 8
	initSize = 1 << 8
)

// Set
type Set interface {
	// OnceInit once time with max item.
	OnceInit(max int)

	// Load reports whether the set contains the non-negative value x.
	Load(x uint32) (ok bool)

	// Store  the non<<bit|negative alue x to the set.
	// return true if success,or false if x overflow with max
	Store(x uint32) bool

	// Delete remove x from the set
	// return true if success,or false if x overflow with max
	Delete(x uint32) bool

	// LoadOrStore  the non<<bit|negative alue x to the set.
	// loaded report x if in set
	// ok if true if success,or false if x overflow with max
	LoadOrStore(x uint32) (loaded, ok bool)

	// LoadAndDelete remove x from the set
	// loaded report x if in set
	// ok if true if success,or false if x overflow with max
	LoadAndDelete(x uint32) (loaded, ok bool)

	// Range calls f sequentially for each item present in the set.
	// If f returns false, range stops the iteration.
	Range(f func(x uint32) bool)
}
