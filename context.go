package endly

import (
	"github.com/viant/toolbox"
	"fmt"
	"github.com/viant/endly/common"
	"net/url"
	"github.com/viant/toolbox/storage"
	"strings"
)

//TODO Execution detail Tracking of all run (time taken, request, response)

var converter = toolbox.NewColumnConverter("yyyy-MM-dd HH:ss")

var serviceManagerKey = (*serviceManager)(nil)
var deferFunctionsKey = (*[]func())(nil)
var stateKey = (*common.Map)(nil)

type Resource struct {
	Name           string
	Version        string
	URL            string
	Type           string
	Credential     string
	CredentialFile string
	ParsedURL      *url.URL
}

func (r *Resource) Session() string {
	result := r.ParsedURL.Hostname() + ":" + r.ParsedURL.Port()
	if r.ParsedURL.User != nil {
		result = r.ParsedURL.User.Username() + "@" + result
	}
	return result;
}

func (r *Resource) LoadCredential() (string, string, error) {
	if r.CredentialFile == "" {
		return "", "", nil
	}

	credential := &storage.PasswordCredential{}
	err := toolbox.LoadConfigFromUrl("file://"+r.CredentialFile, credential)
	if err != nil {
		return "", "", reportError(fmt.Errorf("Failed to auth URL: %v", err))
	}
	return credential.Username, credential.Password, nil
}

func (r *Resource) AuthURL() (string, error) {
	if r.CredentialFile == "" {
		return r.URL, nil
	}

	username, password, err := r.LoadCredential()
	if err != nil {
		return "", err
	}
	return strings.Replace(r.URL, "//", "//"+username+"@"+password, 1), nil
}


type Context struct {
	toolbox.Context
}

func reportError(err error) error {
	fileName, funcName, line:= toolbox.CallerInfo(4)
	return fmt.Errorf("%v at %v:%v -> %v", err, fileName, line, funcName)
}


func (c *Context) ExpandResource(resource *Resource) (*Resource, error) {
	var err error
	if resource == nil {
		return nil, reportError(fmt.Errorf("Resource was empty"))
	}
	if resource.URL == "" {
		return nil, reportError(fmt.Errorf("URL was empty"))
	}
	var result = &Resource{
		URL:            c.Expand(resource.URL),
		Credential:     c.Expand(resource.Credential),
		Name:           c.Expand(resource.Name),
		Version:        resource.Version,
		Type:           c.Expand(resource.Type),
		CredentialFile: c.Expand(resource.CredentialFile),
	}

	result.ParsedURL, err = url.Parse(result.URL)
	if err != nil {
		return nil, reportError(err)
	}
	if result.CredentialFile == "" && result.Credential != "" {
		manager, err := c.ServiceManager()
		if err != nil {
			return nil, reportError(err)
		}
		result.CredentialFile, err = manager.CredentialFile(result.Credential)
		if err != nil {
			return nil, reportError(err)
		}
	}

	return result, nil
}

func (c *Context) ServiceManager() (ServiceManager, error) {
	var manager = &serviceManager{}
	if ! c.GetInto(serviceManagerKey, &manager) {
		return nil, reportError(fmt.Errorf("Failed to lookup ServiceManager"))
	}
	return manager, nil
}

func (c *Context) Sessions() (ClientSessions) {
	var result *ClientSessions
	if ! c.Contains(clientSessionKey) {
		var sessions ClientSessions = make(map[string]*ClientSession)
		result = &sessions
		c.Put(clientSessionKey, result)
	} else {
		c.GetInto(clientSessionKey, &result)
	}
	return *result
}

func (c *Context) Service(name string) (Service, error) {
	manager, err := c.ServiceManager()
	if err != nil {
		return nil, err
	}
	return manager.Service(name)
}

func (c *Context) Deffer(functions ... func()) []func() {
	var result *[]func()
	if ! c.Contains(deferFunctionsKey) {
		var functions = make([]func(), 0)
		result = &functions
		c.Put(deferFunctionsKey, result)
	} else {
		c.GetInto(deferFunctionsKey, &result)
	}

	*result = append(*result, functions...)
	c.Put(deferFunctionsKey, &result)
	return *result
}

func (c *Context) State() common.Map {
	var result *common.Map
	if ! c.Contains(stateKey) {
		aMap := common.NewMap()
		result = &aMap
		c.Put(stateKey, result)
	} else {
		c.GetInto(stateKey, &result)
	}
	return *result
}

func (c *Context) OperatingSystem(sessionName string) *OperatingSystem {
	var sessions = c.Sessions()
	if session, has := sessions[sessionName]; has {
		return session.OperatingSystem
	}
	return nil
}

func (c *Context) Execute(target *Resource, command *ManagedCommand) (*CommandResult, error) {
	if command == nil {
		return nil, nil
	}
	execService, err := c.Service(ExecServiceId)
	if err != nil {
		return nil, err
	}
	commandRequest := NewCommandRequest(target, command)
	response := execService.Run(c, commandRequest)
	if response.Error != nil {
		return nil, response.Error
	}
	if commandResult, ok := response.Response.(*CommandResult); ok {
		return commandResult, nil
	}
	return nil, nil
}

func (c *Context) Transfer(transfers ...*Transfer) (interface{}, error) {
	if transfers == nil {
		return nil, nil
	}
	transferService, err := c.Service(TransferServiceId)
	if err != nil {
		return nil, err
	}
	response := transferService.Run(c, &Transfers{Transfers: transfers})
	if response.Error != nil {
		return nil, response.Error
	}
	return nil, nil
}

func (c *Context) Expand(text string) string {
	state := c.State()
	return Expand(state, text)
}

func (c *Context) AsRequest(serviceName, requestName string, source map[string]interface{}) (interface{}, error) {
	service, err := c.Service(serviceName)
	if err != nil {
		return nil, err
	}
	request, err := service.NewRequest(requestName)
	if err != nil {
		return nil, err
	}
	err = converter.AssignConverted(request, source)
	return request, err
}

func (c *Context) Close() {
	for _, function := range c.Deffer() {
		function()
	}
}
