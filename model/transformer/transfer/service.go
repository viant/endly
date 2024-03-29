package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/url"
	graph "github.com/viant/endly/model/graph"
	"github.com/viant/endly/model/transfer"
	"github.com/viant/endly/model/transformer"
	"github.com/viant/toolbox"
	"path"
	"strings"
)

type Service struct {
	fs    afs.Service
	graph *graph.Service
}

func (s *Service) Transfer(ctx context.Context, URL string, opts ...transformer.Option) (*transfer.Bundle, error) {
	options := transformer.NewOptions(opts...)
	URL = url.Normalize(URL, file.Scheme)
	session := newSession(options, URL)
	workflow, err := s.graph.LoadWorkflow(ctx, URL, options.StorageOptions()...)
	if err != nil {
		return nil, err
	}
	if err := s.transferWorkflow(ctx, URL, session, workflow); err != nil {
		return nil, err
	}

	URI := session.bundle.URI
	if session.options.WithDependencies {
		if err = s.transferDependencies(ctx, session, URI, options); err != nil {
			return nil, err
		}
	}
	if session.options.WithAssets {
		parentURL, _ := url.Split(URL, file.Scheme)
		isBaseURL := url.Equals(session.baseURL, parentURL)
		if !isBaseURL || session.options.IsRoot() {
			if err := s.transferAssets(ctx, parentURL, session); err != nil {
				return nil, err
			}
		}
	}
	return &session.bundle, nil
}

func (s *Service) transferAssets(ctx context.Context, URL string, session *Session) error {

	objects, err := s.fs.List(ctx, URL, session.options.StorageOptions()...)
	if err != nil {
		return err
	}
	for _, object := range objects {
		if url.Equals(object.URL(), URL) {
			continue
		}
		asset := s.newWorkflowAsset(ctx, session, object)
		if asset == nil {
			continue
		}
		if object.IsDir() {
			if err = s.transferAssets(ctx, object.URL(), session); err != nil {
				return err
			}
		}
		session.bundle.Assets = append(session.bundle.Assets, asset)
	}
	return nil
}

func (s *Service) newWorkflowAsset(ctx context.Context, session *Session, object storage.Object) *transfer.Asset {
	var asset *transfer.Asset
	anAsset, isNew, _ := session.assets.LoadAsset(ctx, object.URL())
	if isNew {
		source, _ := s.fs.DownloadWithURL(ctx, object.URL(), session.options.StorageOptions()...)
		asset = &transfer.Asset{
			ID:         session.bundle.Project.ID + "/" + anAsset.URI,
			Location:   anAsset.URI,
			WorkflowID: session.bundle.Workflow.ID,
			IsDir:      anAsset.IsDir(),
			Source:     source,
		}
	}
	return asset
}

func (s *Service) transferDependencies(ctx context.Context, session *Session, URI string, options *transformer.Options) error {
	for _, scheduled := range session.subworkflow {
		subURL := url.Join(session.baseURL, scheduled+".yaml")
		scheduleURI := url.Join(URI, scheduled)
		if URI == "" {
			scheduleURI = scheduled
		}
		subWorkflow, err := s.Transfer(ctx, subURL, options.Options(
			transformer.WithIsRoot(false),
			transformer.WithProjectID(session.bundle.ProjectID),
			transformer.WithAssetsManager(session.assets),
			transformer.WithParentWorkflowID(session.bundle.Workflow.ID),
			transformer.WithBaseURL(session.baseURL),
			transformer.WithURI(scheduleURI))...)
		if err != nil {
			return err
		}
		subWorkflow.Position = len(session.bundle.SubWorkflows)
		session.bundle.SubWorkflows = append(session.bundle.SubWorkflows, subWorkflow)
	}
	return nil
}

func (s *Service) transferWorkflow(ctx context.Context, URL string, session *Session, workflowNode *graph.Node) (err error) {
	asset := session.assets.LoadWorkflow(ctx, URL)
	workflow := session.newWorkflow(workflowNode, asset)
	workflow.ParentId = session.options.ParentWorkflowID
	if workflow.Init, err = workflowNode.Variables("init"); err != nil {
		return err
	}
	if workflow.Post, err = workflowNode.Variables("post"); err != nil {
		return err
	}
	err = s.transferTasks(ctx, session, "", workflow, workflowNode)
	session.bundle.Workflow = workflow
	return err
}

func (s *Service) transferTasks(ctx context.Context, session *Session, parentID string, workflow *transfer.Workflow, workflowNode *graph.Node) error {
	prefix := parentID
	if parentID == "" {
		prefix = workflow.ID
	}
	return workflowNode.Tasks(func(name string, taskNode *graph.Node) error {
		var task *transfer.Task
		switch taskNode.Type {
		case graph.TypeTask:
			taskMap, err := taskNode.TaskMap()
			if err != nil {
				return err
			}
			if task, err = s.newTask(name, taskNode, taskMap, prefix, parentID, session); err != nil {
				return err
			}
			task.Data = taskNode.Data()
			if template := taskNode.Template(); template != nil {
				if err := s.transferTasks(ctx, session, task.ID, workflow, template); err != nil {
					return err
				}
				if err = s.transferTempleExpandable(ctx, session, task, workflow, template); err != nil {
					return err
				}

			} else {
				if err := s.transferTasks(ctx, session, task.ID, workflow, taskNode); err != nil {
					return err
				}
			}
		case graph.TypeAction:
			actionMap, err := taskNode.ActionMap()
			if err != nil {
				return err
			}
			if task, err = s.newTask(name, taskNode, actionMap, prefix, parentID, session); err != nil {
				return err
			}
			task.IsTemplate = taskNode.IsTemplate
			request, err := taskNode.Request()
			if err != nil {
				return err
			}
			if req, ok := request.(map[string]interface{}); ok && len(req) == 1 {
				inputValue, ok := req["input"]
				if !ok {
					inputValue, ok = req["request"]
				}
				if ok {
					if reqTextValue := toolbox.AsString(inputValue); strings.HasPrefix(reqTextValue, "@") {
						task.Input = reqTextValue
						switch task.Service + ":" + task.Action {
						case "workflow:run":
							if task.IsTemplate {
								session.templates = append(session.templates, reqTextValue[1:])

							} else {
								session.subworkflow = append(session.subworkflow, reqTextValue[1:])
							}
						}
					}
				}
			}
			if task.InputURI == "" {
				req, err := json.Marshal(request)
				if err != nil {
					return err
				}
				task.Input = string(req)
			}
		default:
			return fmt.Errorf("unsupported task type: %v", taskNode.Type)
		}
		return nil
	})
}

func (s *Service) newTask(name string, taskNode *graph.Node, aMap map[string]interface{}, prefix string, parentID string, session *Session) (*transfer.Task, error) {
	var err error
	task := &transfer.Task{}
	if err = toolbox.DefaultConverter.AssignConverted(&task, aMap); err != nil {
		return nil, err
	}
	task.SetID(prefix, name)
	if session.options.Instance != nil {
		task.InstanceIndex = session.options.Instance.Index
		task.InstanceTag = session.options.Instance.Tag
	}
	if task.Action != "" {
		if index := strings.Index(task.Action, ":"); index != -1 {
			task.Service = task.Action[:index]
			task.Action = task.Action[index+1:]
		}
	}
	if task.Service == "" {
		task.Service = "workflow"
	}
	task.ParentId = parentID
	task.WorkflowID = session.bundle.Workflow.ID
	if task.Init, err = taskNode.Variables("init"); err != nil {
		return nil, err
	}
	if task.Post, err = taskNode.Variables("post"); err != nil {
		return nil, err
	}
	task.Position = session.taskIndex[task.ParentId]
	session.taskIndex[task.ParentId] = 1 + session.taskIndex[task.ParentId]
	session.bundle.Tasks = append(session.bundle.Tasks, task)

	return task, nil
}

func (s *Service) transferTempleExpandable(ctx context.Context, session *Session, task *transfer.Task, workflow *transfer.Workflow, template *graph.Node) error {
	if task.SubPath == "" {
		return nil
	}
	storageOptions := session.options.StorageOptions()
	templateURL := url.Join(session.baseURL, task.SubPath)
	parent, name := url.Split(templateURL, file.Scheme)
	holder, err := s.fs.Object(ctx, parent, storageOptions...)
	if err != nil {
		return fmt.Errorf("invalid template subpath: %v, %w", parent, err)
	}
	objects, err := s.fs.List(ctx, parent, storageOptions...)
	instances := graph.NewInstances(holder.URL(), name, objects)

	if err = s.loadTemplate(ctx, session, task, instances); err != nil {
		return err
	}

	return nil

}

func (s *Service) loadTemplate(ctx context.Context, session *Session, task *transfer.Task, instances *graph.Instances) error {

	storageOptions := session.options.StorageOptions()
	defaultURL := url.Join(session.baseURL, "default")
	if object, _ := s.fs.Object(ctx, defaultURL, storageOptions...); object != nil {
		anInstance := &graph.Instance{Object: object, Tag: "default"}
		err := s.loadTemplateInstance(ctx, session, task, anInstance)
		if err != nil {
			return err
		}
	}

	for _, instance := range instances.Instances {
		err := s.loadTemplateInstance(ctx, session, task, instance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) loadTemplateInstance(ctx context.Context, session *Session, task *transfer.Task, instance *graph.Instance) error {
	storageOptions := session.options.StorageOptions()
	for _, name := range session.templates {
		candidate := url.Join(instance.Object.URL(), name+".yaml")
		if ok, _ := s.fs.Exists(ctx, candidate, storageOptions...); ok {
			URI := ""
			if index := strings.Index(instance.Object.URL(), session.baseURL); index != -1 {
				URI = url.Join(instance.Object.URL()[1+index+len(session.baseURL):], name)
			}
			bundle, err := s.Transfer(ctx, candidate, session.options.Options(
				transformer.WithProjectID(session.bundle.Project.ID),
				transformer.WithTemplate(name),
				transformer.WithInstance(instance),
				transformer.WithIsRoot(false),
				transformer.WithParentWorkflowID(session.bundle.Workflow.ID),
				transformer.WithAssetsManager(session.assets),
				transformer.WithBaseURL(session.baseURL),
				transformer.WithURI(URI))...,
			)
			bundle.Workflow.Template = task.Tag
			bundle.Workflow.InstanceIndex = instance.Index
			bundle.Workflow.InstanceTag = instance.Tag
			if err != nil {
				return err
			}
			prev := session.bundle.Templates[task.Tag]
			session.bundle.Templates[task.Tag] = append(prev, bundle)
		}
	}
	for _, name := range task.GetData() {
		if strings.HasPrefix(name, "@") {
			name = name[1:]
		}
		dataURL := url.Join(instance.Object.URL(), name)
		s.loadInstanceAsset(ctx, session, task, instance, dataURL)
	}
	return nil
}

func (s *Service) loadInstanceAsset(ctx context.Context, session *Session, task *transfer.Task, instance *graph.Instance, dataURL string) {
	storageOptions := session.options.StorageOptions()
	if dataAsset, _, _ := session.assets.LoadAsset(ctx, dataURL); dataAsset != nil {
		asset := s.newTemplateAsset(ctx, session, task, dataAsset, storageOptions, instance)
		session.bundle.Assets = append(session.bundle.Assets, asset)
		if asset.IsDir {
			if objects, _ := s.fs.List(ctx, dataAsset.URL(), storageOptions...); len(objects) > 0 {
				for _, object := range objects {
					if url.Equals(dataAsset.URL(), object.URL()) {
						continue
					}
					s.loadInstanceAsset(ctx, session, task, instance, object.URL())
				}
			}
		}
	}
}

func (s *Service) newTemplateAsset(ctx context.Context, session *Session, task *transfer.Task, dataAsset *graph.Asset, storageOptions []storage.Option, instance *graph.Instance) *transfer.Asset {
	var source []byte
	if !dataAsset.IsDir() {
		if data, err := s.fs.DownloadWithURL(ctx, dataAsset.URL(), storageOptions...); err == nil {
			source = data
		}
	}
	asset := &transfer.Asset{
		ID:            session.bundle.Workflow.ID + "/" + dataAsset.URI,
		Location:      dataAsset.URI,
		Description:   "",
		WorkflowID:    session.bundle.Workflow.ID,
		IsDir:         dataAsset.IsDir(),
		Template:      task.Tag,
		InstanceIndex: instance.Index,
		InstanceTag:   instance.Tag,
		Position:      0,
		Source:        source,
		Format:        path.Ext(dataAsset.Object.Name()),
		Codec:         "",
	}
	return asset
}

func New() *Service {
	return &Service{fs: afs.New(), graph: graph.New()}
}
