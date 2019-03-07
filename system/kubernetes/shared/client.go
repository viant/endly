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
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
	"path/filepath"
	"strings"
)

var clientKey = (*CtxClient)(nil)

var defaultNamespace = "default"

//CtxClient represents generic google cloud service client
type CtxClient struct {
	CredConfig *cred.Config
	masterURL  string
	cfgContext string
	configPath string
	Namespace  string
	ResetConfig *rest.Config
	clientSet  *kubernetes.Clientset
	RawRequest map[string]interface{}
}


func (c *CtxClient) EndpointIP() string {
	return strings.TrimLeft(c.ResetConfig.Host, "htps:/")
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

	if c.cfgContext != "" {
		c.ResetConfig, err = buildConfigFromFlags(c.cfgContext, configPath)
	} else {
		c.ResetConfig, err = clientcmd.BuildConfigFromFlags(c.masterURL, configPath)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to build conifg %v, %v", configPath, err)
	}
	c.clientSet, err = kubernetes.NewForConfig(c.ResetConfig)
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

//Init get or creates context, client
func Init(context *endly.Context, rawRequest map[string]interface{}) error {
	if len(rawRequest) == 0 {
		return nil
	}
	ctxClient := &CtxClient{}
	if context.Contains(clientKey) {
		context.GetInto(clientKey, &ctxClient)
	}
	ctxClient.RawRequest = rawRequest
	mappings := util.BuildLowerCaseMapping(rawRequest)
	if key, ok := mappings["kubeconfig"]; ok {
		ctxClient.configPath = toolbox.AsString(rawRequest[key])
		ctxClient.clientSet = nil
	}
	if key, ok := mappings["context"]; ok {
		ctxClient.cfgContext = toolbox.AsString(rawRequest[key])
		ctxClient.clientSet = nil
	}
	if ctxClient.Namespace == "" {
		ctxClient.Namespace = defaultNamespace
	}

	if key, ok := mappings["namespace"]; ok && key != "" {
		ctxClient.Namespace = toolbox.AsString(rawRequest[key])
		ctxClient.Namespace = strings.Replace(ctxClient.Namespace, "*", "", 1)
	}

	if key, ok := mappings["masterurl"]; ok {
		ctxClient.masterURL = toolbox.AsString(rawRequest[key])
		ctxClient.cfgContext = ""
		ctxClient.clientSet = nil
	}
	return context.Replace(clientKey, ctxClient)
}
