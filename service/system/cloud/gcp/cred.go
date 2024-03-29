package gcp

import (
	"context"
	"github.com/viant/scy/auth/gcp"
	"github.com/viant/scy/auth/gcp/client"

	"fmt"
	"github.com/go-errors/errors"
	"github.com/viant/endly"
	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"github.com/viant/scy/cred/secret"
	"github.com/viant/toolbox"

	"google.golang.org/api/option"
	htransport "google.golang.org/api/transport/http"
	"net/http"
	"reflect"
	"strings"
)

var configKey = (*gcpCredConfig)(nil)

const userAgent = "endly/e2e"
const DefaultRegion = "us-central1"

type gcpCredConfig struct {
	*cred.Generic
	*scy.Secret
	scopes []string
}

// GetClient creates a new google cloud client.
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
	var options = make([]option.ClientOption, 0)
	options = append(options, option.WithScopes(scopes...))
	isAuth := false
	if credConfig.Secret != nil {
		if data := credConfig.Secret.String(); data != "" {
			options = append(options, option.WithCredentialsJSON([]byte(data)))
			isAuth = true
		}

	}

	var httpClient *http.Client
	if !isAuth {
		gcpService := gcp.New(client.NewScy())
		if httpClient, err = gcpService.AuthClient(context.Background()); err == nil && httpClient != nil {
			options = append(options, option.WithHTTPClient(httpClient))
		}
	}
	var args = make([]interface{}, 1+len(options))
	args[0] = ctx
	for i := range options {
		args[i+1] = options[i]
	}

	output := toolbox.CallFunction(provider, args...)
	if output[1] != nil {
		err := output[1].(error)
		return err
	}
	service := output[0]
	ctxService, ok := reflect.ValueOf(target).Elem().Interface().(CtxClient)
	if !ok {
		return fmt.Errorf("invalid target type: %T", target)
	}

	ctxService.SetCredConfig(credConfig.Generic)
	ctxService.SetHttpClient(httpClient)
	ctxService.SetContext(ctx)
	if err = ctxService.SetService(service); err != nil {
		return err
	}
	return eContext.Replace(key, reflect.ValueOf(target).Elem().Interface())
}

// InitCredentials get or creates aws credential config
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

	config := &gcpCredConfig{Generic: &cred.Generic{}}
	if config.Secret, _ = context.Secrets.Lookup(context.Background(), secret.Resource(secrets.Credentials)); config.Secret != nil {
		config.Generic, _ = config.Secret.Target.(*cred.Generic)
	}
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

func getCredentials(context *endly.Context) (*gcpCredConfig, error) {
	credConfig := &gcpCredConfig{}
	if context.GetInto(configKey, &credConfig) {
		return credConfig, nil
	}
	return nil, fmt.Errorf("gcp credentials not found")
}

func getDefaultClient(ctx context.Context, scopes ...string) (*http.Client, error) {
	o := []option.ClientOption{
		option.WithScopes(scopes...),
		option.WithUserAgent(userAgent),
	}
	httpClient, _, err := htransport.NewClient(ctx, o...)
	return httpClient, err
}
