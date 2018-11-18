package udf

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"log"
	"path"
	"testing"

	"io/ioutil"
)

func TestProtoCodec_AsBinary(t *testing.T) {

	parentDirectory := toolbox.CallerDirectory(3)
	var useCases = []struct {
		description   string
		importPath    string
		protoFile     string
		messageType   string
		binarryLength int
		dataFile      string
	}{
		{
			description:   "message without package",
			importPath:    path.Join(parentDirectory, "test/proto/book"),
			protoFile:     "address_book.proto",
			messageType:   "AddressBook",
			dataFile:      path.Join(parentDirectory, "test/proto/book/data.json"),
			binarryLength: 58,
		},
		{
			description:   "message with package",
			importPath:    path.Join(parentDirectory, "test/proto/pkg"),
			protoFile:     "person.proto",
			messageType:   "foo.Person",
			dataFile:      path.Join(parentDirectory, "test/proto/pkg/data.json"),
			binarryLength: 27,
		},
	}

	for _, useCase := range useCases {

		codec, err := NewProtoCodec(useCase.protoFile, useCase.importPath, useCase.messageType)
		if !assert.Nil(t, err, useCase.description) {
			log.Fatal(err)
		}
		data, err := ioutil.ReadFile(useCase.dataFile)
		if !assert.Nil(t, err, useCase.description) {
			log.Fatal(err)
		}

		binary, err := codec.AsBinary(useCase.messageType, string(data))
		if !assert.Nil(t, err) {
			log.Fatal(err)
		}
		assert.Equal(t, useCase.binarryLength, len(binary), useCase.description)
		converted, err := codec.AsMessage(useCase.messageType, binary)
		if !assert.Nil(t, err, useCase.description) {
			log.Fatal(err)
		}
		assertly.AssertValues(t, string(data), converted, useCase.description)

	}

}
