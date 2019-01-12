package gce

import "fmt"

//CallRequest represents GCE call request, it operates on *cloud.Service client (https://cloud.google.com/compute/docs/reference/latest/)
type CallRequest struct {
	Credentials string        `required:"true" description:"path to secret json file"`                                                                                         //path to secret json file.
	Service     string        `required:"true" description:"field representing service on *compute.Service, for instance Instance field points to  *compute.InstancesService"` //field representing service on *compute.Service, for instance Instance field points to  *compute.InstancesService
	Method      string        `required:"true" description:"method on cloud service, for instance, Get, Start"`                                                                //method on cloud service, for instance, Get, Start
	Parameters  []interface{} `required:"true" description:"actual method parameters"`                                                                                         //actual method parameters
}

//Validate checks if request is valid
func (r *CallRequest) Validate() error {
	if r.Credentials == "" {
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

//CallResponse represents GCE call response
type CallResponse interface{}

