package endly

import _ "github.com/viant/toolbox/storage/scp"

//initialises UDF functions
func init() {
	UdfRegistry["AsTableRecords"] = AsTableRecords
}
