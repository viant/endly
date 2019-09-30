package udf

import "github.com/viant/endly"

//init initialises UDFs functions and register service
func init() {
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})

	endly.UdfRegistry["Dob"] = DateOfBirth
	endly.UdfRegistry["URLJoin"] = URLJoin
	endly.UdfRegistry["URLPath"] = URLPath
	endly.UdfRegistry["Hostname"] = Hostname
	endly.UdfRegistry["GZipper"] = GZipper
	endly.UdfRegistry["GZipContentCorrupter"] = GZipContentCorrupter
	endly.UdfRegistry["AvroReader"] = NewAvroReader

	endly.UdfRegistryProvider["AvroWriter"] = NewAvroWriter
	endly.UdfRegistryProvider["ProtoReader"] = NewProtoReader
	endly.UdfRegistryProvider["ProtoWriter"] = NewProtoWriter
	endly.UdfRegistryProvider["CsvReader"] = NewCsvReader

}
