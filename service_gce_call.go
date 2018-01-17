package endly

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
