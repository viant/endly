package endly

//it uses scp as default transfer protocol
import (
	_ "github.com/viant/endly/static" //load external file to mem storage
	_ "github.com/viant/toolbox/storage/scp"
)

//init initialises UDF functions
func init() {
	UdfRegistry["AsTableRecords"] = AsTableRecords
}
