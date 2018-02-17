package endly

import (
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
)

//DsUnitRegisterRequest represents a register request.
type DsUnitRegisterRequest struct {
	Datastore       string                 `required:"true" description:"datastore name"`
	Config          *dsc.Config            `required:"true" description:"datastore config"` //make sure Deploy.Parameters have database Id key
	Credential      string                 `required:"true" description:"datastore credential file"`
	adminConfig     *dsc.Config            //make sure Deploy.Parameters have database Id key
	AdminDatastore  string                 `description:"admin datastore, needed to connect and create test database"`
	AdminCredential string                 `description:"admin datastore credential file"`
	ClearDatastore  bool                   `description:"flag to re create database"`
	Scripts         []*url.Resource        `description:"URL list with SQL script to run"`
	Tables          []*dsc.TableDescriptor `description:"table descriptor list"`
}

//DsUnitRegisterResponse represents a register response.
type DsUnitRegisterResponse struct {
	Modified int
}

//Init initialises request
func (r *DsUnitRegisterRequest) Init() {
	if r.Config.Parameters == nil {
		r.Config.Parameters = make(map[string]string)
	}
	if r.AdminCredential == "" {
		r.AdminCredential = r.Credential
	}
	if r.AdminDatastore != "" {
		var parameters = make(map[string]string)
		toolbox.CopyMapEntries(r.Config.Parameters, parameters)
		r.adminConfig = &dsc.Config{
			DriverName: r.Config.DriverName,
			Descriptor: r.Config.Descriptor,
			Parameters: parameters,
		}
		r.adminConfig.Parameters["dbname"] = r.AdminDatastore
	}
	if _, exists := r.Config.Parameters["dbname"]; !exists {
		r.Config.Parameters["dbname"] = r.Datastore
	}
}

//Validate check fi request is valid, otherwise returns an error.
func (r *DsUnitRegisterRequest) Validate() error {
	if r.Config == nil {
		return fmt.Errorf("Datastore config was nil")
	}
	return nil
}
