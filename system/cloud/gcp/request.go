package gcp

import (
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

type URLParams map[string][]string

//ExpandMeta expand meta data
func ExpandMeta(context *endly.Context, text string) string {
	gcpCred, err := getCredentials(context)
	if err != nil {
		return text
	}
	if gcpCred.Region == "" {
		gcpCred.Region = DefaultRegion
	}
	state := createCredState(gcpCred)
	return state.ExpandAsText(text)
}

func createCredState(gcpCred *gcpCredConfig) data.Map {
	state := data.NewMap()
	state.SetValue("gcp.projectID", gcpCred.ProjectID)
	state.SetValue("gcp.projectId", gcpCred.ProjectID)
	state.SetValue("gcp.region", gcpCred.Region)
	state.SetValue("gcp.serviceAccount", gcpCred.ClientEmail)
	return state
}

//UpdateActionRequest updates raw request with project, service
func UpdateActionRequest(rawRequest map[string]interface{}, credConfig *gcpCredConfig, client CtxClient) {

	if credConfig.Region == "" {
		credConfig.Region = DefaultRegion
	}
	state := createCredState(credConfig)
	for k, v := range rawRequest {
		rawRequest[k] = state.Expand(v)
	}

	mappings := util.BuildLowerCaseMapping(rawRequest)
	if _, has := mappings["project"]; !has {
		rawRequest["project"] = credConfig.ProjectID
	}
	if _, has := mappings["region"]; !has {
		rawRequest["region"] = credConfig.Region
	}

	var URLParams = make(URLParams)
	if paramsKey, has := mappings["urlparams"]; has {
		params := rawRequest[paramsKey]
		if toolbox.IsMap(params) {
			for k, v := range toolbox.AsMap(params) {
				URLParams[k] = []string{toolbox.AsString(v)}
			}
		}
	}

	rawRequest["urlParams_"] = URLParams
	rawRequest["s"] = client.Service()

}
