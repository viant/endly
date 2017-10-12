package endly

func init() {
	UdfRegistry["AsTableRecords"] = AsTableRecords
	UdfRegistry["AsMap"] = AsMap
	UdfRegistry["AsInt"] = AsInt
}
