package endly

import (
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
)

//DsUnitRegisterRequest represents a register request.
type DsUnitRegisterRequest struct {
	Datastore       string
	Config          *dsc.Config //make sure Deploy.Parameters have database Id key
	Credential      string
	adminConfig     *dsc.Config //make sure Deploy.Parameters have database Id key
	AdminDatastore  string      //Id of admin db
	AdminCredential string
	ClearDatastore  bool
	Scripts         []*url.Resource
	Tables          []*dsc.TableDescriptor
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
