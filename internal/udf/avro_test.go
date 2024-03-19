package udf

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"path"
	"testing"
)

func TestNewAvroWriterProvider(t *testing.T) {

	parent := toolbox.CallerDirectory(3)
	schemaFile := path.Join(parent, "test/avro/user.avsc")

	writer, err := NewAvroWriter(schemaFile, "snappy")
	assert.Nil(t, err)
	type user struct {
		ID   int
		Desc string
	}
	transformed, err := writer(&user{1, "test"}, nil)
	assert.Nil(t, err)
	assert.NotNil(t, transformed)
	output, err := NewAvroReader(transformed, nil)
	assert.Nil(t, err)
	assertly.AssertValues(t, "{\"Desc\":\"test\",\"ID\":1}", output)
}

func TestRegisterProviders(t *testing.T) {
	parent := toolbox.CallerDirectory(3)
	schemaFile := path.Join(parent, "test/avro/user.avsc")

	{
		err := RegisterProviders([]*endly.UdfProvider{
			{
				ID:       "userAvro",
				Provider: "AvroWriter",
				Params: []interface{}{
					schemaFile,
				},
			},
		})
		assert.Nil(t, err)
		_, ok := endly.PredefinedUdfs["userAvro"]
		assert.True(t, ok)
	}
	{
		err := RegisterProviders([]*endly.UdfProvider{
			{
				ID:       "userAvro",
				Provider: "AvroWritaaaaer",
				Params: []interface{}{
					schemaFile,
				},
			},
		})
		assert.NotNil(t, err)
	}
}
