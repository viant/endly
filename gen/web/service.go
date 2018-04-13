package web

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v2"
	"strings"
	"github.com/viant/toolbox/storage"
	"bytes"
	"archive/zip"
)

type Service struct {
	baseTemplateURL string
	baseAssetURL    string
}

func (s *Service) getDbTemplates() ([]*DbTemplate, error) {
	var templateURL = toolbox.URLPathJoin(s.baseTemplateURL, fmt.Sprintf("datastore"))
	templ, err := DownloadAll(templateURL)
	if err != nil {
		return nil, err
	}
	var result = make([]*DbTemplate, 0)
	for k := range templ {
		if strings.HasSuffix(k, "meta.yaml") {
			meta, err := s.loadDbMeta(k, templ)
			if err != nil {
				return nil, err
			}
			result = append(result, &DbTemplate{
				Id:        meta.Id,
				Name:      meta.Name,
				HasConfig: meta.Config != "",
			})
		}
	}
	return result, nil
}

func (s *Service) getAppTemplates() ([]*AppTemplate, error) {
	var templateURL = toolbox.URLPathJoin(s.baseTemplateURL, fmt.Sprintf("app"))
	templ, err := DownloadAll(templateURL)
	if err != nil {
		return nil, err
	}
	var result = make([]*AppTemplate, 0)
	for k := range templ {
		if strings.HasSuffix(k, "meta.yaml") {
			meta, err := s.loadAppMeta(k, templ)
			if err != nil {
				return nil, err
			}
			result = append(result, &AppTemplate{
				Template:    meta.Name,
				Description: meta.Description,
				Sdk:         meta.Sdk,
				Docker:      meta.Docker,
				HasOrigin:   meta.OriginURL != "",
			})
		}
	}
	return result, nil
}

func (s *Service) Get(request *GetRequest) (*GetResponse, error) {
	var response = &GetResponse{
		Status: "ok",
		Sdk:    []string{},
	}
	builder := newBuilder(s.baseTemplateURL)
	sdk, err := builder.NewMapFromURI("sdk/sdk.yaml", nil)
	if err != nil {
		return nil, err
	}
	sdk.Range(func(key string, value interface{}) {
		response.Sdk = append(response.Sdk, fmt.Sprintf("%s:%s", key, toolbox.AsString(value)))
	})

	if response.App, err = s.getAppTemplates(); err != nil {
		return nil, err
	}
	if response.Db, err = s.getDbTemplates(); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *Service) Run(request *RunRequest) (*RunResponse, error) {
	var response = &RunResponse{}
	builder := newBuilder(s.baseTemplateURL)
	err := s.handleDatastore(builder, request.Datastore);
	if err != nil {
		return nil, err
	}

	if err := builder.buildSystem(); err != nil {
		return nil, err
	}
	if err = builder.buildDatastore(); err != nil {
		return nil, err
	}
	err = s.handleBuild(builder, request);
	if err != nil {
		return nil, err
	}
	var destURL = builder.destURL
	destURL = string(destURL[strings.LastIndex(destURL, "/endly"):])
	var writer = new(bytes.Buffer)
	archive := zip.NewWriter(writer)
	if err = storage.Archive(builder.storage, destURL, archive); err != nil {
		return nil, err
	}
	archive.Flush()
	archive.Close()
	response.Data = writer.Bytes()
	return response, err
}

func (s *Service) asData(state data.Map, template interface{}) (map[string]interface{}, error) {
	expanded := state.Expand(template)
	var result = make(map[string]interface{})
	err := yaml.NewDecoder(strings.NewReader(toolbox.AsString(expanded))).Decode(result)
	return result, err
}

func (s *Service) loadDbMeta(URI string, assets map[string]string) (*DbMeta, error) {
	value, ok := assets[URI]
	if ! ok {
		return nil, fmt.Errorf("unable to locate meta %v", assets)
	}
	meta := &DbMeta{}
	err := yaml.NewDecoder(strings.NewReader(toolbox.AsString(value))).Decode(meta)
	return meta, err
}

func (s *Service) loadAppMeta(URI string, assets map[string]string) (*AppMeta, error) {
	value, ok := assets[URI]
	if ! ok {
		return nil, fmt.Errorf("unable to locate meta %v", assets)
	}
	meta := &AppMeta{}
	err := yaml.NewDecoder(strings.NewReader(toolbox.AsString(value))).Decode(meta)
	return meta, err
}

func (s *Service) handleDatastore(builder *builder, datastore *Datastore) error {
	var templateURL = toolbox.URLPathJoin(s.baseTemplateURL, fmt.Sprintf("datastore/%v", datastore.Driver))
	assets, err := DownloadAll(templateURL)
	if err != nil {
		return err
	}
	meta, err := s.loadDbMeta("meta.yaml", assets)
	if err != nil {
		return err
	}
	if err = builder.addDatastoreService(assets, meta, datastore); err != nil {
		return err
	}
	if err = builder.addDatastore(assets, meta, datastore); err != nil {
		return err
	}
	return nil
}

func (s *Service) handleBuild(builder *builder, request *RunRequest) error {
	var build = request.Build
	var templateURL = toolbox.URLPathJoin(s.baseTemplateURL, fmt.Sprintf("app/%v", build.TemplateApp))
	assets, err := DownloadAll(templateURL)
	if err != nil {
		return err
	}

	appMeta, err := s.loadAppMeta("meta.yaml", assets)
	if err == nil {
		err = builder.buildApp(appMeta, request.Build, assets)
	}
	if err == nil {
		err = builder.addSourceCode(appMeta, request.Build, assets)
	}
	if err == nil {
		err = builder.addRegression(appMeta, request)
	}
	if err == nil {
		err = builder.addManager(appMeta, request)
	}
	return err
}

func NewService(baseTemplateURL, baseAssetURL string) *Service {
	return &Service{
		baseTemplateURL: baseTemplateURL,
		baseAssetURL:    baseAssetURL,
	}
}
