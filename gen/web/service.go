package web

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"gopkg.in/yaml.v2"
	"sort"
	"strings"
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
	var result DbTemplates = make([]*DbTemplate, 0)
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
	sort.Sort(result)
	return result, nil
}

func (s *Service) getSdk() ([]string, error) {
	var templateURL = toolbox.URLPathJoin(s.baseTemplateURL, fmt.Sprintf("sdk"))
	templ, err := DownloadAll(templateURL)
	if err != nil {
		return nil, err
	}
	var result = make([]string, 0)
	for k := range templ {
		if strings.HasSuffix(k, "meta.yaml") {
			meta, err := s.loadSdkMeta(k, templ)
			if err != nil {
				return nil, err
			}
			result = append(result, fmt.Sprintf("%v:%v", meta.Sdk, meta.Version))
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
	var result AppTemplates = make([]*AppTemplate, 0)
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
				MultiDb:     meta.MultiDb,
			})
		}
	}
	sort.Sort(result)
	return result, nil
}

func (s *Service) Get(request *GetRequest) (*GetResponse, error) {
	var response = &GetResponse{
		Status: "ok",
		Sdk:    []string{},
	}
	var err error
	if response.Sdk, err = s.getSdk(); err != nil {
		return nil, err
	}
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
	var err error
	var hasSystem = false
	for i, datastore := range request.Datastore {
		err = s.handleDatastore(builder, datastore)
		if err != nil {
			return nil, err
		}
		if builder.dbMeta[i].Service != "" {
			hasSystem = true
		}

	}
	if hasSystem {
		if err := builder.buildSystem(); err != nil {
			return nil, err
		}
	}
	if err = builder.buildDatastore(); err != nil {
		return nil, err
	}
	err = s.handleBuild(builder, request)
	if err != nil {
		return nil, err
	}

	var destURL = builder.destURL
	destURL = string(destURL[strings.LastIndex(destURL, "/e2e"):])
	var writer = new(bytes.Buffer)
	archive := zip.NewWriter(writer)
	if err = storage.Archive(builder.destService, destURL, archive); err != nil {
		return nil, err
	}
	archive.Flush()
	archive.Close()
	//Local debugging
	//err = storage.Copy(builder.destService, destURL, storage.NewFileStorage(), "file:///Projects/go/workspace/zz", nil, nil)
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
	if !ok {
		return nil, fmt.Errorf("unable to locate meta %v", assets)
	}
	meta := &DbMeta{}
	err := yaml.NewDecoder(strings.NewReader(toolbox.AsString(value))).Decode(meta)
	return meta, err
}

func (s *Service) loadSdkMeta(URI string, assets map[string]string) (*SdkMeta, error) {
	value, ok := assets[URI]
	if !ok {
		return nil, fmt.Errorf("unable to locate meta %v", assets)
	}
	meta := &SdkMeta{}
	err := yaml.NewDecoder(strings.NewReader(toolbox.AsString(value))).Decode(meta)
	return meta, err
}

func (s *Service) loadAppMeta(URI string, assets map[string]string) (*AppMeta, error) {
	value, ok := assets[URI]
	if !ok {
		return nil, fmt.Errorf("unable to locate meta %v", assets)
	}
	meta := &AppMeta{}
	err := yaml.NewDecoder(strings.NewReader(toolbox.AsString(value))).Decode(meta)
	meta.hasAppDirectory = hasKeyPrefix("app/", assets)
	return meta, err
}

func (s *Service) handleDatastore(builder *builder, datastore *Datastore) error {
	var templateURL = toolbox.URLPathJoin(s.baseTemplateURL, fmt.Sprintf("datastore/%v", datastore.Driver))
	datastore.Name = strings.TrimSpace(datastore.Name)
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
	if err != nil {
		return err
	}
	if request.Build.Sdk == "" {
		request.Build.Sdk = appMeta.Sdk
	}
	var sdkURL = toolbox.URLPathJoin(s.baseTemplateURL, "sdk")
	sdkAssets, err := DownloadAll(sdkURL)
	if err != nil {
		return err
	}
	var sdk = strings.Split(request.Build.Sdk, ":")[0]
	sdkMeta, err := s.loadSdkMeta(sdk+"/meta.yaml", sdkAssets)
	if err != nil {
		return err
	}

	if request.Origin != "" {
		request.Build.Origin = request.Origin
	}
	if err == nil {
		if request.Build.Sdk == "" {
			request.Build.Sdk = appMeta.Sdk
		}
		if appMeta.AutoDiscovery && request.Origin != "" {
			builder.autoDiscover(request.Build, request.Origin)
		}
		err = builder.buildApp(appMeta, sdkMeta, request, assets)
	}
	if err == nil {
		err = builder.addSourceCode(appMeta, request.Build, assets)
	}
	if err == nil {
		err = builder.addRegression(appMeta, request)
	}
	if err == nil {
		err = builder.addRun(appMeta, request)
	}
	requestJSON, _ := toolbox.AsIndentJSONText(request)
	builder.Upload(".gen", strings.NewReader(requestJSON))
	builder.UploadToEndly(".ver", strings.NewReader(fmt.Sprintf("%v %v\n", endly.AppName, endly.GetVersion())))
	return err
}

func NewService(baseTemplateURL, baseAssetURL string) *Service {
	return &Service{
		baseTemplateURL: baseTemplateURL,
		baseAssetURL:    baseAssetURL,
	}
}
