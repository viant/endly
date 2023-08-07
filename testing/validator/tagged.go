package validator

// TaggedAssert represents tagged with ID assert
type TaggedAssert struct {
	TagID    string
	Expected interface{}
	Actual   interface{}
}
