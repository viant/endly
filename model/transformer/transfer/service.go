package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/transfer"
	"github.com/viant/endly/model/transformer"
	"github.com/viant/endly/model/transformer/graph"
	"github.com/viant/toolbox"
	"strings"
)

type Service struct {
	fs       afs.Service
	internal *graph.Service
}

func (s *Service) Transfer(ctx context.Context, URL string, opts ...transformer.Option) (*transfer.Bundle, error) {
	options := transformer.NewOptions(opts...)
	URL = url.Normalize(URL, file.Scheme)
	session := newSession(options, URL)
	workflow, err := s.internal.LoadWorkflow(ctx, URL, options.StorageOptions()...)
	if err != nil {
		return nil, err
	}
	if err := s.transferWorkflow(ctx, session, workflow); err != nil {
		return nil, err
	}
	URI := session.bundle.URI
	if session.options.WithDependencies {
		for _, scheduled := range session.subworkflow {
			subURL := url.Join(session.baseURL, scheduled+".yaml")
			scheduleURI := url.Join(URI, scheduled)
			if URI == "" {
				scheduleURI = scheduled
			}
			subWorkflow, err := s.Transfer(ctx, subURL, options.Options(
				transformer.WithProjectID(session.bundle.ProjectID),
				transformer.WithURI(scheduleURI))...)
			if err != nil {
				return nil, err
			}
			session.bundle.SubWorkflows = append(session.bundle.SubWorkflows, subWorkflow)
		}
	}
	return &session.bundle, nil
}

func (s *Service) transferWorkflow(ctx context.Context, session *Session, workflowNode *graph.Node) (err error) {
	workflow := session.newWorkflow(workflowNode)
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
			task.Template = taskNode.IsTemplate
			request, err := taskNode.Request()
			if err != nil {
				return err
			}
			if req, ok := request.(map[string]interface{}); ok && len(req) == 1 {
				if reqValue, ok := req["request"]; ok {
					if reqTextValue := toolbox.AsString(reqValue); strings.HasPrefix(reqTextValue, "@") {
						task.RequestURI = reqTextValue
						switch task.Action {
						case "run", "workflow:run", "workflow.run":
							if task.Template {
								session.templates = append(session.templates, reqTextValue[1:])

							} else {
								session.subworkflow = append(session.subworkflow, reqTextValue[1:])
							}
						}
					}
				}
			}
			if task.RequestURI == "" {
				req, err := json.Marshal(request)
				if err != nil {
					return err
				}
				task.Request = string(req)
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
	task.ParentId = parentID
	if task.Init, err = taskNode.Variables("init"); err != nil {
		return nil, err
	}
	if task.Post, err = taskNode.Variables("post"); err != nil {
		return nil, err
	}
	task.Index = session.taskIndex[task.ParentId]
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

	for _, instance := range instances.Instances {
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
					transformer.WithURI(URI))...,
				)
				if err != nil {
					return err
				}
				session.bundle.Templates[task.Tag] = append(session.bundle.Templates[task.ID], bundle)
			}
		}
		fmt.Println(instance.Object.URL())
	}

	return nil

}

func New() *Service {
	return &Service{fs: afs.New(), internal: graph.New()}
}
