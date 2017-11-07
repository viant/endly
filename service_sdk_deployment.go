package endly

import "fmt"

//SdkDeploymentRegisterMeta represents sdk deployment meta
type SystemSdkRegisterMetaDeploymentRequest struct {
	Meta *SdkDeploymentMeta
}


//Validate validates meta request
func (r SystemSdkRegisterMetaDeploymentRequest) Validate() error {
	return nil
}

type SystemSdkRegisterMetaDeploymentResponse struct {
	Sdk string
	SdkVersion string
}


//SdkDeploymentMeta represents a sdk deployment instructions
type SdkDeploymentMeta struct {
	Sdk              string
	SdkVersion       string
	Deployments []*OperatingSystemDeployment
}


func (m *SdkDeploymentMeta) Match(os *OperatingSystem) (*DeploymentDeployRequest, error) {
	for _, candidate := range m.Deployments {
		if  candidate.OsTarget == nil || os.Matches(candidate.OsTarget) {
			return candidate.Deploy, nil
		}
	}
	return nil, fmt.Errorf("Failed to lookup deploymeny for sdk %v %v for os: %v %v %v", m.Sdk, m.SdkVersion, os.Family, os.Name, os.Version)
}


//SdkDeploymentMetaRegistry alias map[string]*SdkDeploymentMeta type
type SdkDeploymentMetaRegistry map[string]*SdkDeploymentMeta

//Register register supplied meta in the registry key is sdk and version
func (r *SdkDeploymentMetaRegistry) Register(meta *SdkDeploymentMeta) {
	var key = meta.Sdk+meta.SdkVersion
	(*r)[key] = meta
}

//Get returns meta for provided sdk, version or error if not found
func (r *SdkDeploymentMetaRegistry) Get(sdk, version string) (*SdkDeploymentMeta, error) {
	var key = sdk+version
	if result, ok := (*r)[key];ok {
		return result, nil
	}
	return nil, fmt.Errorf("Failed to lookup SdkDeploymentMeta for %v %v", sdk,version)
}


var sdkDeploymentMetaRegistry SdkDeploymentMetaRegistry = make(map[string]*SdkDeploymentMeta)


