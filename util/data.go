package util

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/data/udf"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

func AsDataMap(source interface{}) data.Map {
	if source == nil {
		return data.Map(map[string]interface{}{})
	}
	var aMap = make(map[string]interface{})
	if toolbox.IsStruct(source) {
		_ = toolbox.DefaultConverter.AssignConverted(&aMap, source)
	} else if toolbox.IsMap(source) {
		aMap = toolbox.AsMap(source)
	}
	return data.Map(aMap)
}

//AsExtractable returns  text and data structure
func AsExtractable(input interface{}) (string, map[string]interface{}) {
	var extractableOutput string
	var structuredOutput data.Map
	switch value := input.(type) {
	case string:
		extractableOutput = value
	case []byte:
		extractableOutput = string(value)
	case []interface{}:
		if len(value) > 0 {
			if toolbox.IsString(value[0]) {
				extractableOutput = toolbox.AsString(value[0])
			} else {
				structuredOutput = AsDataMap(value[0])
			}
		}
	default:
		structuredOutput = AsDataMap(value)
	}
	if extractableOutput != "" {
		if toolbox.IsStructuredJSON(extractableOutput) {
			if strings.HasPrefix(strings.Trim(extractableOutput, " \r\n"), "[") {
				structuredOutput = data.NewMap()
				if aSlice, err := toolbox.JSONToSlice(extractableOutput); err == nil {
					structuredOutput.Put("value", aSlice)
				}
			} else if aMap, err := toolbox.JSONToMap(extractableOutput); err == nil {
				structuredOutput = data.Map(aMap)
			}
		}
	}
	return extractableOutput, structuredOutput
}

//ListResource returns list of matcher resource URL
func ListResource(baseURLs []string, URI string) ([]string, error) {
	if strings.HasPrefix(URI, "@") {
		URI = string(URI[1:])
	}
	for _, baseURL := range baseURLs {
		parent, matchingExpr := path.Split(URI)
		if parent != "" {
			baseURL = toolbox.URLPathJoin(baseURL, parent)
		} else {
			matchingExpr = URI
		}
		var exprSuffix = ""
		if path.Ext(matchingExpr) == "" {
			exprSuffix = ".+"
		}
		regExprText := strings.Replace(matchingExpr, "*", ".+", strings.Count(matchingExpr, "*"))
		regExprText = regExprText + exprSuffix
		if !strings.HasPrefix(regExprText, ".+") {
			regExprText = ".+" + regExprText
		}
		regExpression := regexp.MustCompile(regExprText)
		resource := url.NewResource(baseURL)
		storageService, err := storage.NewServiceForURL(resource.URL, "")
		if err != nil {
			return nil, err
		}
		objects, err := storageService.List(resource.URL)
		if err != nil {
			continue
		}

		var result = make([]string, 0)
		for _, candidate := range objects {
			if !candidate.IsContent() {
				continue
			}
			if regExpression.MatchString(candidate.URL()) {
				result = append(result, candidate.URL())
			}
		}
		if len(result) > 0 {
			sort.Strings(result)
			return result, nil
		}
	}
	return nil, nil
}

//LoadData load and decode URI into a pointer
func LoadData(baseURLs []string, URI string) (interface{}, error) {
	var result interface{}
	URI = strings.TrimSpace(URI)
	URIs := getURIa(URI)
	mainAssetURI := URIs[0]

	rawContent := ""
	mainResource, err := LoadResourceFromBaseURLs(baseURLs, mainAssetURI, &rawContent)
	if err != nil {
		return nil, err
	}
	matched, err := expandMapWithArgumentsIfMatched(baseURLs, URIs, mainResource, rawContent)
	if err != nil {
		return nil, err
	}
	if matched != nil {
		return matched, nil
	}

	if _, err := LoadResourceFromBaseURLs(baseURLs, mainAssetURI, &result); err != nil {
		return nil, err
	}
	for i := 1; i < len(URIs); i++ {
		var aMap = data.NewMap()
		if _, err := LoadResourceFromBaseURLs(baseURLs, URIs[i], &aMap); err != nil {
			return nil, err
		}
		result = aMap.Expand(result)

	}
	return result, nil
}

//LoadMap load and decode URI into map
func LoadMap(baseURLs []string, URI string) (map[string]interface{}, error) {
	URI = strings.TrimSpace(URI)
	URIs := getURIa(URI)
	mainAssetURI := URIs[0]
	rawContent := ""
	resource, err := LoadResourceFromBaseURLs(baseURLs, mainAssetURI, &rawContent)
	if err != nil {
		return nil, err
	}
	matched, err := expandMapWithArgumentsIfMatched(baseURLs, URIs, resource, rawContent)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if matched != nil {
		result = toolbox.AsMap(matched)
	}

	if len(result) == 0 {
		result = make(map[string]interface{})
		if resource, err = LoadResourceFromBaseURLs(baseURLs, mainAssetURI, &result); err != nil {
			return nil, err
		}
	}

	if len(URIs) > 1 {
		if normalized, err := NormalizeMap(result, true); err == nil {
			result = normalized
		}
	}

	for i := 1; i < len(URIs); i++ {
		var aMap = data.NewMap()
		if resource, err = LoadResourceFromBaseURLs(baseURLs, URIs[i], &aMap); err != nil {
			return nil, err
		}
		udf.Register(aMap)
		result = toolbox.AsMap(aMap.Expand(result))
	}
	if err != nil {
		return nil, err
	}
	if _, has := result["AssetURL"]; !has {
		result["AssetURL"] = resource.URL
	}

	return result, nil
}

func getURIa(URI string) []string {
	if spaceCount := strings.Count(URI, " "); spaceCount > 0 {
		URI = strings.Replace(URI, " ", "|", spaceCount)
	}
	var result = make([]string, 0)
	for _, uri := range strings.Split(URI, "|") {
		if uri == "" {
			continue
		}
		result = append(result, uri)
	}
	return result
}

//If main content has $arg{index} or $args{index}, it will expand with corresponding URIs[index-1]
// the fist one has full content of the resource, the latter  removes the first '{' and the last '}' characters.
func expandMapWithArgumentsIfMatched(baseURLs []string, URIs []string, mainResource *url.Resource, mainContent string) (interface{}, error) {
	if len(URIs) < 2 {
		return nil, nil
	}
	if strings.Contains(mainContent, "$arg") || strings.Contains(mainContent, "${arg") {
		isDataArgument := strings.Contains(mainContent, "argData")
		if isDataArgument {
			return expandArgumentAsData(baseURLs, URIs, mainResource, mainContent)
		}
		return expandArgumentAsLiterals(baseURLs, URIs, mainResource, mainContent)
	}
	return nil, nil
}

func expandArgumentAsData(baseURLs []string, URIs []string, mainResource *url.Resource, mainContent string) (interface{}, error) {
	result, err := decodeResourceContent(mainResource, mainContent)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(URIs); i++ {
		argData, err := LoadData(baseURLs, URIs[i])
		if err != nil {
			return nil, err
		}
		aMap := data.NewMap()
		key := fmt.Sprintf("argData%d", i-1)
		aMap.Put(key, argData)
		result = aMap.Expand(result)
	}
	return result, nil
}

func expandArgumentAsLiterals(baseURLs []string, URIs []string, mainResource *url.Resource, mainContent string) (interface{}, error) {
	var aMap = data.NewMap()
	for i := 1; i < len(URIs); i++ {
		var text = ""
		if _, err := LoadResourceFromBaseURLs(baseURLs, URIs[i], &text); err != nil {
			return nil, err
		}
		aMap.Put(fmt.Sprintf("arg%d", i-1), text)
		trimText := strings.TrimSpace(text)
		if strings.Index(trimText, "{") < strings.Index(trimText, "[") {
			trimText = string(trimText[strings.Index(trimText, "{")+1 : strings.LastIndex(trimText, "}")-1])
		} else if strings.LastIndex(trimText, "]") < len(trimText) &&
			strings.Index(trimText, "[") > 0 && strings.Index(trimText, "[") < strings.LastIndex(trimText, "]") {
			trimText = string(trimText[strings.Index(trimText, "[")+1 : strings.LastIndex(trimText, "]")-1])
		}
		aMap.Put(fmt.Sprintf("args%d", i-1), trimText)

	}
	mainContent = aMap.ExpandAsText(mainContent)
	return decodeResourceContent(mainResource, mainContent)
}

func decodeResourceContent(resource *url.Resource, content string) (interface{}, error) {
	var result interface{}
	err := resource.DecoderFactory().Create(strings.NewReader(content)).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %v, %v", resource.URL, err)
	}
	return toolbox.NormalizeKVPairs(result)
}

//LoadResourceFromBaseURLs loads resource from base URLs and URI, returns the first successfully loaded resource or error
func LoadResourceFromBaseURLs(baseURLs []string, URI string, result interface{}) (*url.Resource, error) {
	var err error
	var resource *url.Resource

	for _, baseURL := range baseURLs {
		resource, err = LoadResource(baseURL, URI, result)
		if err == nil {
			return resource, nil
		}
		if IsNotSuchResourceError(err) {
			continue
		}
		return nil, err
	}
	if IsNotSuchResourceError(err) {
		var resourceURLs = []string{}
		var reported = make(map[string]bool)
		for _, baseURL := range baseURLs {
			for _, ext := range []string{".json", ".yaml", ".yml", ".txt", ""} {
				candidate := toolbox.URLPathJoin(baseURL, (string(URI[1:]) + ext))
				if _, ok := reported[candidate]; ok {
					continue
				}
				reported[candidate] = true
				resourceURLs = append(resourceURLs, candidate)
			}
		}
		return nil, &NotSuchResourceError{Message: "failed to locate one of the following: " + strings.Join(resourceURLs, ",\n\t")}
	}
	return nil, err
}

//LoadResource load and decode URI into result pointer
func LoadResource(baseURL, URI string, result interface{}) (*url.Resource, error) {
	if !strings.HasPrefix(URI, "@") {
		return nil, fmt.Errorf("expected @ prefix but had: %v\n", URI)
	}
	URI = string(URI[1:])
	if baseURL != "" && (!strings.Contains(URI, ":/") && !strings.HasPrefix(URI, "/")) {
		URI = toolbox.URLPathJoin(baseURL, URI)
	}
	resource := url.NewResource(URI)
	var loadError, err error
	for _, ext := range []string{".json", ".yaml", ".yml", ".txt", ""} {
		resource := url.NewResource(URI + ext)
		if stringPointer, ok := result.(*string); ok {
			text := ""
			text, err = resource.DownloadText()
			if err == nil {
				*stringPointer = text
				return resource, nil
			}
		} else {
			if err = resource.Decode(result); err == nil {
				return resource, nil
			}
		}
		err = ClassifyErrorIfMatched(err)
		if !IsNotSuchResourceError(err) {
			loadError = err
		}
	}
	if loadError == nil {
		loadError = err
	}
	return resource, loadError
}
