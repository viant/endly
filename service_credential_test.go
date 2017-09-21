package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
	"github.com/viant/toolbox"
	"path"
	"os"
)

func TestNewCredentialService(t *testing.T) {
	manager := endly.NewManager()
	service, err := manager.Service(endly.CredentialServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	context := manager.NewContext(toolbox.NewContext())


	var mysecret = path.Join(os.Getenv("HONE"), "secret/mysql")
	if toolbox.FileExists(mysecret) {
		response := service.Run(context, &endly.CredentailSetRequest{
			Aliases: map[string]string{
				"mysql": path.Join(os.Getenv("HONE"), "secret/mysql.json"),
			},
		})
		assert.Equal(t, "", response.Error)
	}

	response := service.Run(context, &endly.CredentailSetRequest{
		Aliases: map[string]string{
			"mysql": path.Join(os.Getenv("HONE"), "foaerasdfdas.json"),
		},
	})
	assert.True(t, response.Error != "")
}