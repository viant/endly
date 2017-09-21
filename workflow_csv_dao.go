package endly

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"path"
	"strings"
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
	scanner := bufio.NewScanner(strings.NewReader(content))
	workflowMap, err := d.load(context, resource, scanner)
	if err != nil {
		return nil, err
	}
	var result = &Workflow{
		source: source,
	}
	err = converter.AssignConverted(result, workflowMap)
	return result, err
}

//processTag create data structure in the result map, it also check if the reference for tag was used before unless it is the first tag (result tag)
func (d *WorkflowDao) processTag(context *Context, tag *Tag, result common.Map, isResultKey bool, deferredReferences map[string]func(value interface{})) error {
	if result.Has(tag.Name) {
		return nil
	}
	if tag.IsArray {
		var collection = common.NewCollection()
		result.Put(tag.Name, collection)
		setter, has := deferredReferences[tag.Name]
		if !has {
			return fmt.Errorf("Missing reference %v in the previous rows", tag.Name)
		}
		setter(collection)

	} else {
		var object = make(map[string]interface{})
		result.Put(tag.Name, object)
		if !isResultKey {
			setter, has := deferredReferences[tag.Name]
			if !has {
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
		return fmt.Errorf("Failed to import workflow: %v %v", resourceDetail, err)
	}
	workflow, err := d.Load(context, resource)
	if err != nil {
		return fmt.Errorf("Failed to import workflow: %v %v", resourceDetail, err)
	}
	manager, err := context.Manager()
	if err != nil {
		return err
	}
	service, err := manager.Service(WorkflowServiceId)
	if err != nil {
		return err
	}
	serviceResponse := service.Run(context, &WorkflowRegisterRequest{
		Workflow: workflow,
	})
	if serviceResponse.Error != "" {
		return errors.New(serviceResponse.Error)
	}
	if err != nil {
		return fmt.Errorf("Failed to import workflow: %v %v", resourceDetail, err)
	}
	return nil
}

type FieldExpression struct {
	expression        string
	Field             string
	Child             *FieldExpression
	IsArray           bool
	HasSubPath        bool
	HasArrayComponent bool
	IsRoot            bool
}

func (f *FieldExpression) Set(value interface{}, target common.Map, indexes ...int) {
	var index = 0
	if !target.Has(f.Field) {
		if f.IsArray {
			target.Put(f.Field, common.NewCollection())
		} else if f.HasSubPath {
			target.Put(f.Field, common.NewMap())
		}
	}

	var data common.Map

	var action func(data common.Map, indexes ...int)
	if !f.HasSubPath {
		if f.IsArray {
			action = func(data common.Map, indexes ...int) {
				collection := target.GetCollection(f.Field)
				(*collection)[index] = value
			}
		} else {
			action = func(data common.Map, indexes ...int) {
				var isValueSet = false
				if data.Has(f.Field) {
					existingValue := data.Get(f.Field)
					if toolbox.IsMap(existingValue) && toolbox.IsMap(value) { //is existing value is a map and existing value is map
						// then add keys to existing map
						existingMap := data.GetMap(f.Field)
						existingMap.Apply(toolbox.AsMap(value))
						isValueSet = true
					} else if toolbox.IsSlice(existingValue) {//is existing value is a slice append elements
						existingSlice := data.GetCollection(f.Field)
						if toolbox.IsSlice(value) {
							for _, item := range toolbox.AsSlice(value) {
								existingSlice.Push(item)
							}
						} else {
							existingSlice.Push(value)
						}
						data.Put(f.Field, existingSlice)
						isValueSet = true
					}
				}
				if ! isValueSet {
					data.Put(f.Field, value)
				}
			}
		}

	} else {
		action = func(data common.Map, indexes ...int) {
			f.Child.Set(value, data, indexes...)
		}
	}

	if f.IsArray {
		index, indexes = shiftIndex(indexes...)
		collection := target.GetCollection(f.Field)
		collection.ExpandWithMap(index + 1)
		data, _ = (*collection)[index].(common.Map)

	} else if f.HasSubPath {
		data = target.GetMap(f.Field)
	} else {
		data = target
	}
	action(data, indexes...)
}

func shiftIndex(indexes ...int) (int, []int) {
	var index int
	if len(indexes) > 0 {
		index = indexes[0]
		indexes = indexes[1:]
	}
	return index, indexes
}

func NewFieldExpression(expression string) *FieldExpression {

	isRoot := strings.HasPrefix(expression, "/")
	if isRoot {
		expression = string(expression[1:])
	}
	var result = &FieldExpression{
		expression:        expression,
		HasArrayComponent: strings.Contains(expression, "[]"),
		IsArray:           strings.HasPrefix(expression, "[]"),
		HasSubPath:        strings.Contains(expression, "."),
		Field:             expression,
		IsRoot:            isRoot,
	}
	if result.HasSubPath {
		dotPosition := strings.Index(expression, ".")
		result.Field = string(result.Field[:dotPosition])
		result.Child = NewFieldExpression(string(expression[dotPosition+1:]))
	}
	if result.IsArray {
		result.Field = string(result.Field[2:])
	}

	return result
}

//processHeaderLine extract from line a tag from column[0], add deferredRefences for a tag, decodes fields from remaining column,
func (d *WorkflowDao) processHeaderLine(context *Context, result common.Map, decoder toolbox.Decoder, resultTag string, deferredReferences map[string]func(value interface{})) (*toolbox.DelimiteredRecord, *Tag, string, error) {
	record := &toolbox.DelimiteredRecord{Delimiter: ","}
	err := decoder.Decode(record)
	if err != nil {
		return nil, nil, "", err
	}
	tag := NewTag(record.Columns[0])
	var isResultTag = resultTag == ""
	if isResultTag {
		resultTag = tag.Name
	}
	err = d.processTag(context, tag, result, isResultTag, deferredReferences)
	if err != nil {
		return nil, nil, "", err
	}
	return record, tag, resultTag, nil
}

func (d *WorkflowDao) load(context *Context, resource *Resource, scanner *bufio.Scanner) (map[string]interface{}, error) {
	var result = common.NewMap()
	var record *toolbox.DelimiteredRecord
	var deferredReferences = make(map[string]func(value interface{}))
	var referenceUsed = make(map[string]bool)
	var resultTag string
	var tag *Tag
	var object common.Map
	var rootObject common.Map
	var err error
	var lines = make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	for i := 0; i < len(lines); i++ {
		var recordHeight = 0
		line := lines[i]
		if strings.HasPrefix(line, "import") {
			err := d.importWorkflow(context, resource, strings.TrimSpace(string(line[5:])))
			if err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(line, "//") {
			continue
		}
		isHeaderLine := isHeaderByte(line[0])
		decoder := d.factory.Create(strings.NewReader(line))
		if isHeaderLine {
			record, tag, resultTag, err = d.processHeaderLine(context, result, decoder, resultTag, deferredReferences)
			if err != nil {
				return nil, err
			}
			if rootObject == nil {
				rootObject = result.GetMap(resultTag)
			}
			continue
		}
		if tag == nil {
			continue
		}

		record.Record = make(map[string]interface{})
		err := decoder.Decode(record)
		if err != nil {
			return nil, err
		}
		if record.IsEmpty() {
			continue
		}
		object = getObject(tag, result)
		for j := 1; j < len(record.Columns); j++ {
			fieldExpressions := record.Columns[j]
			if fieldExpressions == "" {
				continue
			}

			value, has := record.Record[fieldExpressions]
			if !has || value == nil || toolbox.AsString(value) == "" {
				continue
			}
			field := NewFieldExpression(fieldExpressions)
			textValue := toolbox.AsString(value)
			val, isReference, err := d.normalizeValue(context, resource, textValue)
			if err != nil {
				return nil, err
			}
			if isReference {
				referenceKey := toolbox.AsString(val)
				deferredReferences[referenceKey] = func(reference interface{}) {
					object.Put(fieldExpressions, reference)
					referenceUsed[referenceKey] = true
				}
			} else {
				if field.IsRoot {
					field.Set(val, rootObject)
					if field.HasArrayComponent {
						recordHeight = d.setArrayValues(field, i, lines, record, fieldExpressions, rootObject, recordHeight)
					}

				} else {

					field.Set(val, object)
					if field.HasArrayComponent {
						recordHeight = d.setArrayValues(field, i, lines, record, fieldExpressions, object, recordHeight)
					}
				}
			}
		}
		i += recordHeight
	}
	err = checkeUnsuedReferences(referenceUsed, deferredReferences)
	if err != nil {
		return nil, err
	}
	var workflowObject = result.GetMap(resultTag)
	return workflowObject, nil
}

func checkeUnsuedReferences(referenceUsed map[string]bool, deferredReferences map[string]func(value interface{})) error {
	for k := range referenceUsed {
		delete(deferredReferences, k)
	}
	if len(deferredReferences) == 0 {
		return nil
	}
	var pendingKeys = make([]string, 0)
	for k := range deferredReferences {
		pendingKeys = append(pendingKeys, k)
	}
	return fmt.Errorf("Unresolved references: %v", strings.Join(pendingKeys, ","))
}

func getObject(tag *Tag, result common.Map) common.Map {
	var data common.Map
	if tag.IsArray {
		data = common.NewMap()
		result.GetCollection(tag.Name).Push(data)
	} else {
		data = result.GetMap(tag.Name)
	}
	return data
}

func (d *WorkflowDao) setArrayValues(field *FieldExpression, i int, lines []string, record *toolbox.DelimiteredRecord, fieldExpressions string, data common.Map, recordHeight int) int {
	if field.HasArrayComponent {
		var itemCount = 0

		for k := i + 1; k < len(lines); k++ {

			if (! strings.HasPrefix(lines[k], ",")) {
				break
			}
			arrayValueDecoder := d.factory.Create(strings.NewReader(lines[k]))
			arrayItemRecord := &toolbox.DelimiteredRecord{
				Columns:   record.Columns,
				Delimiter: record.Delimiter,
			}
			arrayValueDecoder.Decode(arrayItemRecord)
			itemValue := arrayItemRecord.Record[fieldExpressions]
			if itemValue == nil || toolbox.AsString(itemValue) == ""  {
				break
			}
			itemCount++
			field.Set(itemValue, data, itemCount)
		}
		if recordHeight < itemCount {
			recordHeight = itemCount
		}
	}
	return recordHeight
}

func isHeaderByte(b byte) bool {
	return (b >= 65 && b <= 93) || (b >= 97 && b <= 122)
}


func (d *WorkflowDao) getExternalResource(context *Context, resource *Resource, resourceDetail string) (*Resource, error) {
	var URL, credential string
	if strings.Contains(resourceDetail, ",") {
		var pair = strings.Split(resourceDetail, ",")
		URL = strings.TrimSpace(pair[0])
		credential = strings.TrimSpace(pair[1])
	} else if strings.Contains(resourceDetail, "://") {
		URL = resourceDetail
	} else if strings.HasPrefix(resourceDetail, "/") {
		URL = fmt.Sprintf("file://%v", resourceDetail)
	} else {
		parent, _ := path.Split(resource.ParsedURL.Path)
		URL = string(resource.URL[:strings.Index(resource.URL, "://")]) + fmt.Sprintf("://%v", path.Join(parent, resourceDetail))
		credential = resource.Credential
	}
	return &Resource{
		URL:            URL,
		Credential: credential,
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
			return nil, false, fmt.Errorf("Failed to decode: %v %v", value, err)
		}
		return jsonObject, false, nil
	case jsonArrayPrefix:
		var jsonArray = make([]interface{}, 0)
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(value)).Decode(&jsonArray)
		if err != nil {
			return nil, false, fmt.Errorf("Failed to decode: %v %v", value, err)
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
