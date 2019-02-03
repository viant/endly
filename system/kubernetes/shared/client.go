package shared

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
)

var clientKey = (*CtxClient)(nil)

//CtxClient represents generic google cloud service client
type CtxClient struct {
	ApiVersion string
	CredConfig *cred.Config
	masterURL  string
	cfgContext string
	configPath string
	clientSet  *kubernetes.Clientset
}

func (c *CtxClient) ConfigPath() string {
	if c.configPath != "" {
		return c.configPath
	}
	c.configPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	return c.configPath
}

func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

func (c *CtxClient) Clientset() (*kubernetes.Clientset, error) {
	if c.clientSet != nil {
		return c.clientSet, nil
	}
	var err error
	configPath := c.ConfigPath()
	var config *rest.Config
	if c.cfgContext != "" {
		config, err = buildConfigFromFlags(c.cfgContext, configPath)
	} else {
		config, err = clientcmd.BuildConfigFromFlags(c.masterURL, configPath)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to build conifg %v, %v", configPath, err)
	}
	c.clientSet, err = kubernetes.NewForConfig(config)
	return c.clientSet, err
}

//GetCtxClient get or creates a new  kubernetess client.
func GetCtxClient(context *endly.Context) (*CtxClient, error) {
	client := &CtxClient{}
	if context.Contains(clientKey) {
		if context.GetInto(clientKey, &client) {
			return client, nil
		}
	}
	err := context.Replace(clientKey, client)
	return client, err
}

//InitClient get or creates client
func InitClient(context *endly.Context, rawRequest map[string]interface{}) error {
	if len(rawRequest) == 0 {
		return nil
	}
	client := &CtxClient{}
	if context.Contains(clientKey) {
		context.GetInto(clientKey, &client)
	}
	mappings := util.BuildLowerCaseMapping(rawRequest)
	if key, ok := mappings["kubeconfig"]; ok {
		client.configPath = toolbox.AsString(rawRequest[key])
		client.clientSet = nil
	}
	if key, ok := mappings["context"]; ok {
		client.cfgContext = toolbox.AsString(rawRequest[key])
		client.clientSet = nil
	}
	if key, ok := mappings["masterurl"]; ok {
		client.masterURL = toolbox.AsString(rawRequest[key])
		client.cfgContext = ""
		client.clientSet = nil
	}
	if key, ok := mappings["apiversion"]; ok {
		client.ApiVersion = toolbox.ToCaseFormat(toolbox.AsString(rawRequest[key]), toolbox.CaseLowerCamel, toolbox.CaseUpperCamel)
	} else {
		client.ApiVersion = "V1"
	}
	return nil
}
