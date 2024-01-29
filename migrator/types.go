package migrator

type postmanType int64

const (
	requests postmanType = iota
	environment
	globals
	notPostman
)

type postmanObject struct {
	nodes      map[string]interface{}
	objectType postmanType
}
