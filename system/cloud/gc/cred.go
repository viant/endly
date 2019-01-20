package gc

import (
	"context"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"golang.org/x/oauth2"
	"google.golang.org/api/gensupport"
	"reflect"
	"strings"
)

var configKey = (*gcCredConfig)(nil)

type gcCredConfig struct {
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

	var credConfig *gcCredConfig
	if eContext.Contains(configKey) {
		credConfig = &gcCredConfig{}
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
	context.Background()
	jwtConfig, err := credConfig.NewJWTConfig(scopes...)
	if err != nil {
		return err
	}
	ctx := context.Background()
	httpClient := oauth2.NewClient(ctx, jwtConfig.TokenSource(ctx))
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
	ctxService.SetContext(ctx)
	if err = ctxService.SetService(service); err != nil {
		return err
	}
	return eContext.Replace(key, reflect.ValueOf(target).Elem().Interface())
}

//InitCredentials get or creates aws credential config
func InitCredentials(context *endly.Context, rawRequest map[string]interface{}) (*gcCredConfig, error) {
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
			credConfig := &gcCredConfig{}
			if context.GetInto(configKey, &credConfig) {
				return credConfig, nil
			}
		}
		return nil, fmt.Errorf("unable to locate Google API client, credentials attribute was empty")
	}
	credConfig, err := context.Secrets.GetCredentials(secrets.Credentials)
	if err != nil {
		return nil, err
	}

	config := &gcCredConfig{Config: credConfig}
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
	return config, err
}

//UpdateActionRequest updates raw request with project, service
func UpdateActionRequest(rawRequest map[string]interface{}, config *gcCredConfig, client CtxClient) {
	for v, key := range []string{"Project"} {
		if _, has := rawRequest[key]; has {
			rawRequest["project"] = v
			delete(rawRequest, key)
		}
	}
	if _, has := rawRequest["project"]; !has {
		rawRequest["project"] = config.ProjectID
	}

	var URLParams = make(gensupport.URLParams)
	if urlParams, ok := rawRequest["urlParams"]; ok {
		if toolbox.IsMap(urlParams) {
			for k, v := range toolbox.AsMap(urlParams) {
				URLParams[k] = []string{toolbox.AsString(v)}
			}
		}
	}
	rawRequest["urlParams_"] = URLParams
	rawRequest["s"] = client.Service()
}
