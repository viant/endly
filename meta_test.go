package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
)

func TestMetaService_Lookup(t *testing.T) {
	manager := endly.NewManager()
	meta := endly.NewMetaService()
	var services = endly.Services(manager)
	for _, service := range services {

		for _, action := range service.Actions() {
			_, err := meta.Lookup(service.ID(), action)
			assert.Nil(t, err)
		}
		_, err := meta.Lookup(service.ID(), "abc")
		assert.NotNil(t, err)
	}

}
