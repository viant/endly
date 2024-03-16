package storage_test

import (
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/system/storage/copy"

	"log"
)

func ExampleCopy() {
	request := storage.NewCopyRequest(nil, copy.New(location.NewResource("/tmp/folde"), location.NewResource("s3://mybucket/data", "aws-e2e"), false, true, nil))
	response := &storage.CopyResponse{}
	err := endly.Run(nil, request, response)
	if err != nil {
		log.Fatal(err)
	}
}
