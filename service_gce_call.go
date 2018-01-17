package endly

import "fmt"

//GCECallRequest represents GCE call request, it operates on *cloud.Service client (https://cloud.google.com/compute/docs/reference/latest/)
type GCECallRequest struct {
	Credential string        //path to secret json file.
	Service    string        //field representing service on *compute.Service, for instance Instance field points to  *compute.InstancesService
	Method     string        //method on cloud service, for instance, Get, Start
	Parameters []interface{} //actual method parameters
}

//GCECallResponse represents GCE call response
type GCECallResponse struct {
	Error    string
	Response interface{}
}

//Validate checks if request is valid
func (r *GCECallRequest) Validate() error {
	if r.Credential == "" {
		return fmt.Errorf("credentials were empty for GCE %v.%v", r.Service, r.Method)
	}
	if r.Service == "" {
		return fmt.Errorf("service was empty for GCE %v", r.Method)
	}
	if r.Method == "" {
		return fmt.Errorf("method was empty for GCE %v", r.Service)
	}
	return nil
}
