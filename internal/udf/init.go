package udf

import (
	"github.com/viant/endly"
	"github.com/viant/endly/internal/udf/buildin"
)

// init initialises UDFs functions and register service
func init() {
	_ = endly.Registry.Register(func() endly.Service {
		return New()
	})
	endly.PredefinedUdfs["LoadData"] = LoadData
	endly.PredefinedUdfs["Dob"] = DateOfBirth
	endly.PredefinedUdfs["URLJoin"] = URLJoin
	endly.PredefinedUdfs["URLPath"] = URLPath
	endly.PredefinedUdfs["Hostname"] = Hostname
	endly.PredefinedUdfs["GZipper"] = GZipper
	endly.PredefinedUdfs["GZipContentCorrupter"] = GZipContentCorrupter
	endly.PredefinedUdfs["AvroReader"] = NewAvroReader

	endly.UdfRegistryProvider["AvroWriter"] = NewAvroWriter
	endly.UdfRegistryProvider["ProtoReader"] = NewProtoReader
	endly.UdfRegistryProvider["ProtoWriter"] = NewProtoWriter
	endly.UdfRegistryProvider["CsvReader"] = NewCsvReader

	endly.PredefinedUdfs["IsJSON"] = buildin.IsJSON
	endly.PredefinedUdfs["WorkingDirectory"] = buildin.WorkingDirectory
	endly.PredefinedUdfs["Pwd"] = buildin.WorkingDirectory
	endly.PredefinedUdfs["HasResource"] = buildin.HasResource
	endly.PredefinedUdfs["Md5"] = buildin.Md5
	endly.PredefinedUdfs["Zip"] = buildin.Zip
	endly.PredefinedUdfs["Unzip"] = buildin.Unzip
	endly.PredefinedUdfs["UnzipText"] = buildin.UnzipText
	endly.PredefinedUdfs["Markdown"] = buildin.Markdown
	endly.PredefinedUdfs["Cat"] = buildin.Cat
	endly.PredefinedUdfs["LoadBinary"] = buildin.LoadBinary
	endly.PredefinedUdfs["AssetsToMap"] = buildin.AssetsToMap
	endly.PredefinedUdfs["BinaryAssetsToMap"] = buildin.BinaryAssetsToMap
	endly.PredefinedUdfs["CurrentHour"] = buildin.CurrentHour
	endly.PredefinedUdfs["MatchAnyRow"] = buildin.MatchAnyRow

}
