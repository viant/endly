package udf

import (
	"github.com/viant/endly"
	"github.com/viant/endly/udf/buildin"
)

// init initialises UDFs functions and register service
func init() {
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})

	endly.UdfRegistry["LoadData"] = LoadData
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

	endly.UdfRegistry["IsJSON"] = buildin.IsJSON
	endly.UdfRegistry["WorkingDirectory"] = buildin.WorkingDirectory
	endly.UdfRegistry["Pwd"] = buildin.WorkingDirectory
	endly.UdfRegistry["HasResource"] = buildin.HasResource
	endly.UdfRegistry["Md5"] = buildin.Md5
	endly.UdfRegistry["Zip"] = buildin.Zip
	endly.UdfRegistry["Unzip"] = buildin.Unzip
	endly.UdfRegistry["UnzipText"] = buildin.UnzipText
	endly.UdfRegistry["Markdown"] = buildin.Markdown
	endly.UdfRegistry["Cat"] = buildin.Cat
	endly.UdfRegistry["LoadBinary"] = buildin.LoadBinary
	endly.UdfRegistry["AssetsToMap"] = buildin.AssetsToMap
	endly.UdfRegistry["BinaryAssetsToMap"] = buildin.BinaryAssetsToMap
	endly.UdfRegistry["CurrentHour"] = buildin.CurrentHour
	endly.UdfRegistry["MatchAnyRow"] = buildin.MatchAnyRow

}
