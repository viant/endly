package gcp

import (
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"google.golang.org/api/gensupport"
)

//ExpandMeta expand meta data
func ExpandMeta(context *endly.Context, text string) string {
	gcpCred, err := getCredentials(context)
	if err != nil {
		return text
	}
	if gcpCred.Region == "" {
		gcpCred.Region = DefaultRegion
	}
	state := data.NewMap()
	state.SetValue("gcp.projectID", gcpCred.ProjectID)
	state.SetValue("gcp.region", gcpCred.Region)
	return state.ExpandAsText(text)
}

//UpdateActionRequest updates raw request with project, service
func UpdateActionRequest(rawRequest map[string]interface{}, config *gcpCredConfig, client CtxClient) {
	state := data.NewMap()

	if config.Region == "" {
		config.Region = DefaultRegion
	}

	state.SetValue("gcp.projectID", config.ProjectID)
	state.SetValue("gcp.region", config.Region)
	for k, v := range rawRequest {
		rawRequest[k] = state.Expand(v)
	}

	mappings := util.BuildLowerCaseMapping(rawRequest)
	if _, has := mappings["project"]; !has {
		rawRequest["project"] = config.ProjectID
	}
	if _, has := mappings["region"]; !has {
		rawRequest["region"] = config.Region
	}

	var URLParams = make(gensupport.URLParams)
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
