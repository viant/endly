package endly

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"path"
	"strings"
)

var internalReferencePrefix = []byte("%")[0]

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
	var result = &Workflow{}
	err = converter.AssignConverted(result, workflowMap)
	result.source = source
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
	asset := strings.TrimSpace(source)
	resource, err := d.getExternalResource(context, resource, "", asset)
	if err != nil {
		return fmt.Errorf("Failed to import workflow: %v %v", asset, err)
	}
	workflow, err := d.Load(context, resource)
	if err != nil {
		return fmt.Errorf("Failed to import workflow: %v %v", asset, err)
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
		return fmt.Errorf("Failed to import workflow: %v %v", asset, err)
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
					if toolbox.IsMap(existingValue) && toolbox.IsMap(value) { //is existing Value is a map and existing Value is map
						// then add keys to existing map
						existingMap := data.GetMap(f.Field)
						existingMap.Apply(toolbox.AsMap(value))
						isValueSet = true
					} else if toolbox.IsSlice(existingValue) { //is existing value is a slice append elements
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
				if !isValueSet {
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
	var subPath = ""
	var rootObject common.Map
	var err error
	var lines = make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	for i := 0; i < len(lines); i++ {
		var recordHeight = 0
		line := lines[i]

		var hasActiveIterator = tag.HasActiveIterator()
		if hasActiveIterator {
			state := context.State()
			state.Put("index", tag.Iterator.Index())
			line = d.expandIteratorIndex(context, line)
		}
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
			if hasActiveIterator {
				if tag.Iterator.Next() {
					i = tag.line
					continue
				}
			}
			record, tag, resultTag, err = d.processHeaderLine(context, result, decoder, resultTag, deferredReferences)
			tag.line = i
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


		if relativeSubPath, ok := record.Record["Subpath"]; ok {
			subPath = toolbox.AsString(relativeSubPath)
		}

		object = getObject(tag, result)
		if hasActiveIterator {
			object["Group"]= tag.Iterator.Index()
		}


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

			val, isReference, err := d.normalizeValue(context, resource, subPath, textValue)
			if err != nil {
				return nil, fmt.Errorf("Failed to normalizeValue: %v, %v", textValue, err)
			}
			if isReference {
				referenceKey := toolbox.AsString(val)
				deferredReferences[referenceKey] = func(reference interface{}) {
					object.Put(fieldExpressions, reference)
					referenceUsed[referenceKey] = true
				}
			} else {
				if field.IsRoot {


					if field.HasArrayComponent {
						var expr = strings.Replace(field.expression, "[]", "", 1)
						bucket, has := rootObject.GetValue(expr)
						if ! has {
							bucket = common.NewCollection()
						}
						var bucketSlice = toolbox.AsSlice(bucket)
						if toolbox.IsSlice(val) {
							aSlice := toolbox.AsSlice(val)
							for _, item := range aSlice {
								bucketSlice = append(bucketSlice, item)
							}
						} else {
							bucketSlice = append(bucketSlice, val)
						}
						rootObject.SetValue(expr, bucketSlice)
					} else {
						field.Set(val, rootObject)
					}
					//if field.HasArrayComponent {
					//	recordHeight, err = d.setArrayValues(field, i, lines, record, fieldExpressions, rootObject, recordHeight, resource, context, subPath)
					//}

				} else {

					field.Set(val, object)
					if field.HasArrayComponent {
						recordHeight, err = d.setArrayValues(field, i, lines, record, fieldExpressions, object, recordHeight, resource, context, subPath)
					}
				}
				if err != nil {
					return nil, err
				}
			}
		}

		if _, has := object["SubPath"];!has {
			object["SubPath"] = subPath
		}
		i += recordHeight

		if i+1 == len(lines) && hasActiveIterator {
			if tag.Iterator.Next() {
				i = tag.line
				continue
			}

		}

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

func (d *WorkflowDao) setArrayValues(field *FieldExpression, i int, lines []string, record *toolbox.DelimiteredRecord, fieldExpressions string, data common.Map, recordHeight int, resource *Resource, context *Context, subpath string) (int, error) {
	if field.HasArrayComponent {
		var itemCount = 0

		for k := i + 1; k < len(lines); k++ {

			if !strings.HasPrefix(lines[k], ",") {
				break
			}
			arrayValueDecoder := d.factory.Create(strings.NewReader(lines[k]))
			arrayItemRecord := &toolbox.DelimiteredRecord{
				Columns:   record.Columns,
				Delimiter: record.Delimiter,
			}
			arrayValueDecoder.Decode(arrayItemRecord)
			itemValue := arrayItemRecord.Record[fieldExpressions]
			if itemValue == nil || toolbox.AsString(itemValue) == "" {
				break
			}
			itemCount++
			val, _, err := d.normalizeValue(context, resource, subpath, toolbox.AsString(itemValue))
			if err != nil {
				return 0, err
			}
			field.Set(val, data, itemCount)
		}
		if recordHeight < itemCount {
			recordHeight = itemCount
		}
	}
	return recordHeight, nil
}

func isHeaderByte(b byte) bool {
	return (b >= 65 && b <= 93) || (b >= 97 && b <= 122)
}

func (d *WorkflowDao) getExternalResource(context *Context, resource *Resource, subpath, asset string) (*Resource, error) {
	if asset == "" {
		return nil, reportError(fmt.Errorf("Resource was empty"))
	}
	if strings.HasPrefix(asset, "#") {
		asset = string(asset[1:])
	}
	var URL, credential string
	var useSubpath = false
	if strings.Contains(asset, ",") {
		var pair = strings.Split(asset, ",")
		URL = strings.TrimSpace(pair[0])

		credential = strings.TrimSpace(pair[1])
	} else if strings.Contains(asset, "://") {
		URL = asset
	} else if strings.HasPrefix(asset, "/") {
		URL = fmt.Sprintf("file://%v", asset)
	} else {

		parent, _ := path.Split(resource.ParsedURL.Path)
		if subpath != "" {
			useSubpath = true
			fileCandidate := path.Join(parent, subpath, asset)
			if toolbox.FileExists(fileCandidate) {
				URL = fmt.Sprintf("file://%v", fileCandidate)
			}
		}

		if URL == "" {
			URL = string(resource.URL[:strings.Index(resource.URL, "://")]) + fmt.Sprintf("://%v", path.Join(parent, asset))
		}
		service, err := storage.NewServiceForURL(URL, credential)
		if err != nil {
			return nil, err
		}
		if exists, _ := service.Exists(URL); !exists {
			endlyResource, err := NewEndlyRepoResource(context, asset)
			if err == nil {
				if exists, _ := service.Exists(endlyResource.URL); exists {
					URL = endlyResource.URL
				}
			} else if useSubpath {
				fileCandidate := path.Join(parent, subpath, asset)
				URL = fmt.Sprintf("file://%v", fileCandidate)
			}
		}
		credential = resource.Credential
	}
	return &Resource{
		URL:        URL,
		Credential: credential,
	}, nil
}





func (d *WorkflowDao) loadMap(context *Context, parentResource *Resource, subpath, asset string, escapeQuotes bool, index int) (common.Map, error) {
	var aMap = make(map[string]interface{})
	var assetContent = asset
	if strings.HasPrefix(asset, "#") {
		resource, err := d.getExternalResource(context, parentResource, subpath, asset)
		if err != nil {
			return nil, err
		}
		assetContent, err = resource.DownloadText()
		if err != nil {
			return nil, err
		}
	}


	assetContent = d.expandIteratorIndex(context, assetContent)
	assetContent = strings.Trim(assetContent, " \t\n\r")
	if strings.HasPrefix(assetContent, "{")  {
		err:= toolbox.NewJSONDecoderFactory().Create(strings.NewReader(assetContent)).Decode(&aMap)
		if err != nil {
			return nil, err
		}
	}

	if escapeQuotes {
		for k, v := range aMap {
			if toolbox.IsMap(v) || toolbox.IsSlice(v) {
				buf := new(bytes.Buffer)
				err := toolbox.NewJSONEncoderFactory().Create(buf).Encode(v)
				if err != nil {
					return nil, err
				}
				v = buf.String()
			}
			if toolbox.IsString(v) {
				textValue := toolbox.AsString(v)
				if strings.Contains(textValue, "\"") {
					textValue = strings.Replace(textValue, "\\", "\\\\", len(textValue))

					textValue = strings.Replace(textValue, "\n", "", len(textValue))

					textValue = strings.Replace(textValue, "\"", "\\\"", len(textValue))
					//fmt.Printf(textValue)

					aMap[k] = textValue

				}
			}
		}
	}
	aMap[fmt.Sprintf("arg%v", index)] = assetContent
	aMap[fmt.Sprintf("args%v", index)] = string(assetContent[1:len(assetContent)-1])
	return common.Map(aMap), nil
}

func (d *WorkflowDao) loadExternalResource(context *Context, parentResource *Resource, subpath, asset string) (string, error) {

	resource, err := d.getExternalResource(context, parentResource, subpath, asset)
	var result string
	if err == nil {
		result, err = resource.DownloadText()
	}
	if err != nil {
		return "", fmt.Errorf("Failed to load external resource: %v %v", asset, err)
	}
	return result, err
}

func getUdfIfDefined(expression string) (func(interface{}, common.Map) (interface{}, error), string, error) {
	if !strings.HasPrefix(expression, "!") {
		return nil, expression, nil
	}
	startArgumentPosition := strings.Index(expression, "(")
	endArgumentPosition := strings.LastIndex(expression, ")")
	if startArgumentPosition != -1 && endArgumentPosition > startArgumentPosition {
		udfName := string(expression[1:startArgumentPosition])
		var has bool
		udfFunction, has := UdfRegistry[udfName]
		if !has {
			var available = toolbox.MapKeysToStringSlice(UdfRegistry)
			return nil, "", fmt.Errorf("Failed to lookup udf function %v on %v, avaialbe:[%v]", udfName, expression, strings.Join(available, ","))
		}
		value := string(expression[startArgumentPosition+1: endArgumentPosition])
		return udfFunction, value, nil
	}
	return nil, expression, nil
}

func (d *WorkflowDao) contextualizeValue(context *Context, value string) (interface{}, bool, error) {
	if len(value) == 0 {
		return nil, false, nil
	}

	switch value[0] {
	case internalReferencePrefix:
		return string(value[1:]), true, nil
	case jsonObjectPrefix:
		var jsonObject = make(map[string]interface{})
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(value)).Decode(&jsonObject)
		if err != nil {
			return nil, false, fmt.Errorf("Failed to decode: %v %T, %v", value, value, err)
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


func  (d *WorkflowDao) expandIteratorIndex(context *Context, data string) string {
	var state = context.State()
	var index = state.GetString("index")
	data = strings.Replace(data, "${index}", toolbox.AsString(index), len(data))
	data = strings.Replace(data, "$index", toolbox.AsString(index), len(data))
	return data
}


func (d *WorkflowDao) normalizeValue(context *Context, parentResource *Resource, subpath, value string) (interface{}, bool, error) {
	//TODO refactor to simplify and extend functionaliy
	var err error
	var state = context.State()

	if strings.HasPrefix(value, "#") {
		var assets = strings.Split(value, "|")
		mainAsset, err := d.loadExternalResource(context, parentResource, subpath, assets[0])
		if err != nil {
			return nil, false, err
		}
		mainAsset = strings.TrimSpace(mainAsset)
		mainAsset = d.expandIteratorIndex(context, mainAsset)

		escapeQuotes := strings.HasPrefix(mainAsset, "{") || strings.HasPrefix(mainAsset, "[")
		for i := 1; i < len(assets); i++ {
			aMap, err := d.loadMap(context, parentResource, subpath, assets[i], escapeQuotes, i-1)
			if err != nil {
				return nil, false, err
			}
			mainAsset = ExpandAsText(aMap, mainAsset)
		}
		value = mainAsset
	}
	udfFunction, value, err := getUdfIfDefined(value)
	if err != nil {
		return nil, false, err
	}
	resultValue, isReference, err := d.contextualizeValue(context, value)
	if udfFunction != nil {
		resultValue, err = udfFunction(resultValue, state)
	}

	return resultValue, isReference, err
}

func NewWorkflowDao() *WorkflowDao {
	return &WorkflowDao{
		factory: toolbox.NewDelimiterDecoderFactory(),
	}
}

type TagIterator struct {
	Template string
	Min      int
	Max      int
	index    int
}

func (i *TagIterator) Has() bool {
	return i.index <= i.Max
}

func (i *TagIterator) Next() bool {
	i.index++
	return i.Has()
}

func (i *TagIterator) Index() string {
	return fmt.Sprintf(i.Template, i.index)
}

type Tag struct {
	Name     string
	IsArray  bool
	Iterator *TagIterator
	line     int
}

func (t *Tag) HasActiveIterator() bool {
	if t == nil {
		return false
	}
	return t.Iterator != nil && t.Iterator.Has()
}

func NewTag(key string) *Tag {
	var result = &Tag{
		Name: key,
	}
	key = decodeIteratrIfPresent(key, result)
	if string(key[0:2]) == "[]" {
		result.Name = string(key[2:])
		result.IsArray = true
	}

	return result
}
func decodeIteratrIfPresent(key string, result *Tag) string {
	iteratorStartPosition := strings.Index(key, "{")
	if iteratorStartPosition != -1 {
		iteratorEndPosition := strings.Index(key, "}")
		if iteratorEndPosition != -1 {
			iteratorConstrain := key[iteratorStartPosition+1:iteratorEndPosition]
			pair := strings.Split(iteratorConstrain, "..")
			for i, value := range pair {
				pair[i] = strings.TrimSpace(value)
			}
			if len(pair) == 2 {
				result.Iterator = &TagIterator{
					Min:      toolbox.AsInt(pair[0]),
					Max:      toolbox.AsInt(pair[1]),
					Template: "%0" + toolbox.AsString(len(pair[1])) + "d",
				}
				result.Iterator.index = result.Iterator.Min
				key = string(key[:iteratorStartPosition])
			}
		}
	}
	return key
}
