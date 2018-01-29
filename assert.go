package endly

import "github.com/viant/assertly"


//Assert validates expected against actual
func Assert(context *Context, root string, expected, actual interface{}) (*assertly.Validation, error) {
	ctx := assertly.NewDefaultContext()
	ctx.Context = context.Context
	var rootPath = assertly.NewDataPath(root)
	return assertly.AssertWithContext(expected, actual, rootPath, ctx)
}
