package compare

// Result is the outcome of a comparison result between 2 values
// Defined as byte for minimal memory consumption
type Result byte

const (
	// Equal represents equality of the current value to other value
	// e.g. 1 == 1
	Equal Result = iota + 1
	// BothNulls represents a case in which the value and the other value
	// are both nulls, e.g. null == null
	BothNulls
	// Null represents a case in which only 1 of the values is null
	// e.g. 1 == null
	Null
	// Greater represents greater than other value
	// e.g. 2 > 1
	Greater
	// Less represents smaller value than other value
	// e.g. 1 < 2
	Less
)
