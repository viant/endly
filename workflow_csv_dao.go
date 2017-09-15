package endly

import (
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
	"fmt"
	"bufio"
	"path"
	"github.com/pkg/errors"
)

var internalReferencePrefix = []byte("%")[0]
var externalReferencePrefix = []byte("#")[0]
var jsonObjectPrefix = []byte("{")[0]
var jsonArrayPrefix = []byte("[")[0]

type WorkflowDao struct {
	factory toolbox.DecoderFactory
}



func (d *WorkflowDao) Load(context *Context, source *Resource) (*Workflow, error) {
	resource, err := context.ExpandResource(source)
	if err != nil {
		return nil, err
	}
	content, err := resource.DownloadText()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(context.Expand(content)))
	workflowMap, err := d.load(context, resource, scanner)
	if err != nil {
		return nil, err
	}
	var result = &Workflow{}
	err = converter.AssignConverted(result, workflowMap)
	return result, err
}

func (d *WorkflowDao) processTag(context *Context, tag *Tag, result common.Map, isResultKey bool, deferredReferences map[string]func(value interface{})) error {
	if result.Has(tag.Name) {
		return nil
	}
	if tag.IsArray {
		var collection = common.NewCollection()
		result.Put(tag.Name, collection)
		setter, has := deferredReferences[tag.Name]
		if ! has {
			return fmt.Errorf("Missing reference %v in the previous rows", tag.Name)
		}
		setter(collection)

	} else {
		var object = make(map[string]interface{})
		result.Put(tag.Name, object)
		if !isResultKey {
			setter, has := deferredReferences[tag.Name]
			if ! has {
				return fmt.Errorf("Missing reference %v in the previous rows", tag.Name)
			}
			setter(object)
		}
	}
	return nil
}

func (d *WorkflowDao) importWorkflow(context *Context, resource *Resource, source string) error {
	resourceDetail := strings.TrimSpace(source)
	resource, err := d.getExternalResource(context, resource, resourceDetail)
	if err != nil {
		return  fmt.Errorf("Failed to import workflow: %v %v", resourceDetail,  err)
	}
	workflow, err :=d.Load(context, resource)
	if err != nil {
		return fmt.Errorf("Failed to import workflow: %v %v", resourceDetail,  err)
	}
	manager ,err := context.Manager()
	if err != nil {
		return err
	}
	service, err := manager.Service(WorkflowServiceId)
	if err != nil {
		return err
	}
	serviceResponse := service.Run(context, &WorkflowRegisterRequest{
		Workflow:workflow,
	})
	if serviceResponse.Error != "" {
		return errors.New(serviceResponse.Error)
	}
	if err != nil {
		return  fmt.Errorf("Failed to import workflow: %v %v", resourceDetail,  err)
	}
	return nil
}


func (d *WorkflowDao) load(context *Context, resource *Resource, scanner *bufio.Scanner) (map[string]interface{}, error) {
	var result = common.NewMap()
	var record *toolbox.DelimiteredRecord
	var deferredReferences = make(map[string]func(value interface{}))
	var referenceUsed = make(map[string]bool)
	var resultKey string
	var tag *Tag
	var data common.Map
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "import") {
			err := d.importWorkflow(context, resource, strings.TrimSpace(string(line[5:])))
			if err != nil {
				return nil, err
			}
			continue
		}
		isHeaderLine := isLetter(line[0])
		decoder := d.factory.Create(strings.NewReader(line))
		if isHeaderLine {
			record = &toolbox.DelimiteredRecord{
				Delimiter: ",",
			}
			err := decoder.Decode(record)
			if err != nil {
				return nil, err
			}
			tag = NewTag(record.Columns[0])
			var isResultKey = resultKey == "";
			if isResultKey {
				resultKey = tag.Name
			}
			err = d.processTag(context, tag, result, isResultKey, deferredReferences)
			if err != nil {
				return nil, err
			}
			continue
		}

		if tag == nil {
			continue
		}
		if tag.IsArray {
			data = common.NewMap()
			result.GetCollection(tag.Name).Push(data)
		} else {
			data = result.GetMap(tag.Name)
		}
		err := decoder.Decode(record)
		if err != nil {
			return nil, err
		}
		for i := 1; i < len(record.Columns); i++ {
			column := record.Columns[i]
			if column == "" {
				continue
			}
			value, has := record.Record[column];
			if !has || value == nil ||  toolbox.AsString(value) == "" {
				continue
			}
			textValue := toolbox.AsString(value)
			val, isReference, err := d.normalizeValue(context, resource, textValue)
			if err != nil {
				return nil, err
			}

			if isReference {
				referenceKey := toolbox.AsString(val)
				deferredReferences[referenceKey] = func(reference interface{}) {
					data.Put(column, reference)
					referenceUsed[referenceKey] = true
				}
			} else {
				data.Put(column, val)
			}
		}
	}
	for k, _ := range referenceUsed {
		delete(deferredReferences, k)
	}
	if len(deferredReferences) > 0 {
		var pendingKeys = make([]string, 0)
		for k, _ := range deferredReferences {
			pendingKeys = append(pendingKeys, k)
		}
		return nil, fmt.Errorf("Unresolved references: %v", strings.Join(pendingKeys, ","))
	}
	var workflowObject = result.GetMap(resultKey)
	return workflowObject, nil
}


func isLetter(b byte) bool {
	return (b >= 65 && b <= 93) || (b >= 97 && b <= 122 )
}

func (d *WorkflowDao) getExternalResource(context *Context, resource *Resource, resourceDetail string) (*Resource, error) {
	var URL, credentailFile string
	if strings.Contains(resourceDetail, ",") {
		var pair= strings.Split(resourceDetail, ",")
		URL = strings.TrimSpace(pair[0])
		credentailFile = strings.TrimSpace(pair[1])
	} else if strings.Contains(resourceDetail, "://") {
		URL = resourceDetail
	} else if strings.HasPrefix(resourceDetail, "/") {
		URL = fmt.Sprintf("file://%v", resourceDetail)
	} else {
		parent, _ := path.Split(resource.ParsedURL.Path)
		URL = string(resource.URL[:strings.Index(resource.URL, "://")]) + fmt.Sprintf("://%v", path.Join(parent, resourceDetail))
		credentailFile = resource.CredentialFile
	}
	return &Resource{
		URL:            URL,
		CredentialFile: credentailFile,
	}, nil
}


func (d *WorkflowDao) normalizeValue(context *Context, resource *Resource, value string) (interface{}, bool, error) {

	if value[0] == externalReferencePrefix {
		resource, err := d.getExternalResource(context, resource, string(value[1:]))
		if err == nil {
			value, err = resource.DownloadText()
		}
		if err != nil {
			return nil, false, err
		}
	}

	switch value[0] {
	case internalReferencePrefix:
		return string(value[1:]), true, nil
	case jsonObjectPrefix:
		var jsonObject = make(map[string]interface{})
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(value)).Decode(&jsonObject)
		if err != nil {
			return nil, false, err
		}
		return jsonObject, false, nil
	case jsonArrayPrefix:
		var jsonArray = make([]map[string]interface{}, 0)
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(value)).Decode(&jsonArray)
		if err != nil {
			return nil, false, err
		}
		return jsonArray, false, nil

	}
	return value, false, nil
}

func NewWorkflowDao() *WorkflowDao {
	return &WorkflowDao{
		factory: toolbox.NewDelimiterDecoderFactory(),
	}
}

type Tag struct {
	Name    string
	IsArray bool
}

func NewTag(key string) *Tag {
	var name = key
	var isArray = false
	if string(name[0:2]) == "[]" {
		name = string(key[2:])
		isArray = true

	}
	return &Tag{
		Name:    name,
		IsArray: isArray,
	}
}
