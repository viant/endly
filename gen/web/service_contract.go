package web

import (
	"fmt"
)

type AppTemplate struct {
	Template    string `json:"template"`
	Description string `json:"description"`
	Sdk         string `json:"sdk"`
	Docker      bool   `json:"docker"`
	HasOrigin   bool   `json:"hasOrigin"`
	MultiDb     bool   `json:"multiDb"`
}

type AppTemplates []*AppTemplate

func (a AppTemplates) Len() int {
	return len(a)
}
func (a AppTemplates) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a AppTemplates) Less(i, j int) bool {
	return a[i].Template < a[j].Template
}

type DbTemplate struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	HasConfig bool   `json:"hasConfig"`
}

type DbTemplates []*DbTemplate

func (a DbTemplates) Len() int {
	return len(a)
}

func (a DbTemplates) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a DbTemplates) Less(i, j int) bool {
	return a[i].Id < a[j].Id
}

//Tag represent a docker tag
type Tag struct {
	Username string
	Registry string
	Image    string
	Version  string
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
	REST           bool
	HTTP           bool
	Selenium       bool   `json:"selenium"`
	UseCaseData    string `json:"useCaseData"`
	DataValidation bool   `json:"dataValidation"`
	LogValidation  bool   `json:"logValidation"`
}

type Build struct {
	Sdk           string
	App           string
	Origin        string
	TemplateApp   string
	Docker        bool
	Dockerfile    bool
	DockerCompose bool
	Tag           *Tag
	path          string
}

type SystemService struct {
	Name    string
	Service string
}

type Datastore struct {
	Driver            string
	Name              string
	Version           string
	Config            bool
	MultiTableMapping bool
}

type RunRequest struct {
	Origin    string
	Build     *Build
	Datastore []*Datastore
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
