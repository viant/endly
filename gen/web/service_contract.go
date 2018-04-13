package web

import "fmt"

type AppTemplate struct {
	Template    string `json:"template"`
	Description string `json:"description"`
	Sdk         string `json:"sdk"`
	Docker      bool   `json:"docker"`
	HasOrigin   bool   `json:"hasOrigin"`
}

type DbTemplate struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	HasConfig bool   `json:"hasConfig"`
}

type GetRequest struct{}
type GetResponse struct {
	Status string
	Error  string
	Sdk    []string       `json:"sdk"`
	App    []*AppTemplate `json:"app"`
	Db     []*DbTemplate  `json:"db"`
}

type Testing struct {
	REST        bool
	HTTP        bool
	Selenium    bool `json:"selenium"`
	UseCaseData bool `json:"useCaseData"`
}

type Build struct {
	Sdk         string
	App         string
	Origin      string
	TemplateApp string
	Docker      bool
	path        string
}

type SystemService struct {
	Name    string
	Service string
}

type Datastore struct {
	Driver  string
	Name    string
	Version string
	Config  bool
}

type RunRequest struct {
	Origin    string
	Build     *Build
	Datastore *Datastore
	Testing   *Testing
}

func (r *RunRequest) Validate() error {
	if r.Build == nil {
		return fmt.Errorf("build was empty")
	}
	if r.Build.Sdk == "" {
		return fmt.Errorf("build.sdk was empty")
	}
	if r.Build.App == "" {
		return fmt.Errorf("build.app was empty")
	}
	return nil
}

type RunResponse struct {
	Data []byte
}
