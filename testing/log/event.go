package log

//RecordAssert represents log record assert
type RecordAssert struct {
	TagID    string
	Expected interface{}
	Actual   interface{}
}
