package gcp

import (
	"context"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/data"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gensupport"
	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
	"net/http"
	"reflect"
	"strings"
)

var configKey = (*gcpCredConfig)(nil)

const userAgent = "endly/e2e"
const defaultRegion = "us-central1"

type gcpCredConfig struct {
	*cred.Config
	scopes []string
}

//GetClient creates a new google cloud client.
func GetClient(eContext *endly.Context, provider, key interface{}, target interface{}, defaultScopes ...string) error {
	if eContext.Contains(key) {
		if eContext.GetInto(key, target) {
			return nil
		}
	}
	var err error
	var credConfig *gcpCredConfig
	if eContext.Contains(configKey) {
		credConfig = &gcpCredConfig{}
		eContext.GetInto(configKey, &credConfig)
	}
	if credConfig == nil {
		return errors.New("unable to locate google API credentials")
	}
	scopes := credConfig.scopes
	if len(scopes) == 0 {
		scopes = defaultScopes
	}
	if len(scopes) == 0 {
		return errors.New("scopes were empty")
	}

	ctx := context.Background()

	if credConfig.ProjectID == "" {
		credentials, err := google.FindDefaultCredentials(ctx, scopes...)
		if err != nil {
			return err
		}
		credConfig.ProjectID = credentials.ProjectID
	}

	var httpClient *http.Client

	if credConfig.ClientEmail != "" {
		jwtConfig, err := credConfig.NewJWTConfig(scopes...)
		if err != nil {
			return err
		}
		httpClient = oauth2.NewClient(ctx, jwtConfig.TokenSource(ctx))
	} else {
		if httpClient, err = getDefaultClient(ctx, scopes...); err != nil {
			return err
		}
	}
	output := toolbox.CallFunction(provider, httpClient)
	if output[1] != nil {
		err := output[1].(error)
		return err
	}
	service := output[0]
	ctxService, ok := reflect.ValueOf(target).Elem().Interface().(CtxClient)
	if !ok {
		return fmt.Errorf("invalid target type: %T", target)
	}
	ctxService.SetCredConfig(credConfig.Config)
	ctxService.SetHttpClient(httpClient)
	ctxService.SetContext(ctx)
	if err = ctxService.SetService(service); err != nil {
		return err
	}
	return eContext.Replace(key, reflect.ValueOf(target).Elem().Interface())
}

//InitCredentials get or creates aws credential config
func InitCredentials(context *endly.Context, rawRequest map[string]interface{}) (*gcpCredConfig, error) {
	if len(rawRequest) == 0 {
		return nil, fmt.Errorf("request was empty")
	}

	secrets := &struct {
		Credentials string
	}{}
	if err := toolbox.DefaultConverter.AssignConverted(secrets, rawRequest); err != nil {
		return nil, err
	}
	if secrets.Credentials == "" {
		if context.Contains(configKey) {
			credConfig := &gcpCredConfig{}
			if context.GetInto(configKey, &credConfig) {
				return credConfig, nil
			}
		}
	}

	credConfig, err := context.Secrets.GetCredentials(secrets.Credentials)
	if err != nil {
		credConfig = &cred.Config{}
	}

	config := &gcpCredConfig{Config: credConfig}
	if scopes, ok := rawRequest["scopes"]; ok {
		if toolbox.IsString(scopes) {
			config.scopes = strings.Split(toolbox.AsString(scopes), ",")
		} else if toolbox.IsSlice(scopes) {
			aSlice := toolbox.AsSlice(scopes)
			config.scopes = make([]string, 0)
			for _, item := range aSlice {
				config.scopes = append(config.scopes, toolbox.AsString(item))
			}
		}
	}
	_ = context.Replace(configKey, config)
	return config, nil
}

func getDefaultClient(ctx context.Context, scopes ...string) (*http.Client, error) {
	o := []option.ClientOption{
		option.WithScopes(scopes...),
		option.WithUserAgent(userAgent),
	}
	httpClient, _, err := htransport.NewClient(ctx, o...)
	return httpClient, err
}

//UpdateActionRequest updates raw request with project, service
func UpdateActionRequest(rawRequest map[string]interface{}, config *gcpCredConfig, client CtxClient) {
	state := data.NewMap()
	state.SetValue("gcp.projectID", config.ProjectID)
	for k, v := range rawRequest {
		rawRequest[k] = state.Expand(v)
	}
	if config.Region == "" {
		config.Region = defaultRegion
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
