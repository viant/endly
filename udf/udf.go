package udf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
)

//TransformWithUDF transform payload with provided UDFs name.
func TransformWithUDF(context *endly.Context, udfName, source string, payload interface{}) (interface{}, error) {
	var state = context.State()
	var udf, has = endly.UdfRegistry[udfName]
	if !has {
		udf, has = getUdfFromContext(udfName, state)
	}
	if !has {
		return nil, fmt.Errorf("failed to lookup udf: %v for: %v", udfName, source)
	}
	transformed, err := udf(payload, state)
	if err != nil {
		return nil, fmt.Errorf("failed to run udf: %v, %v", udfName, err)
	}
	return transformed, nil
}

// Helper to get UDFs from context state
func getUdfFromContext(udfName string, state data.Map) (func(interface{}, data.Map) (interface{}, error), bool) {
	if candidate, has := state[udfName]; has {
		udf, ok := candidate.(func(source interface{}, state data.Map) (interface{}, error))
		return udf, ok
	}
	return nil, false
}

//DateOfBirth returns formatted date of birth, it take  desired age,  [month], [day], [timeformat]
func DateOfBirth(source interface{}, state data.Map) (interface{}, error) {
	if !toolbox.IsSlice(source) {
		return nil, fmt.Errorf("expected slice but had: %T %v", source, source)
	}
	return toolbox.NewDateOfBirthrovider().Get(toolbox.NewContext(), toolbox.AsSlice(source)...)
}

//URLJoin joins base URL and URI path
func URLJoin(source interface{}, state data.Map) (interface{}, error) {
	if !toolbox.IsSlice(source) {
		return nil, fmt.Errorf("expected slice but had: %T %v", source, source)
	}
	var args = toolbox.AsSlice(source)
	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 arguments  but had: %v", len(args))
	}
	var baseURL = strings.Trim(toolbox.AsString(args[0]), " '\"")
	var URI = strings.Trim(toolbox.AsString(args[1]), " '\"")
	return toolbox.URLPathJoin(baseURL, URI), nil
}

//URLPath return path from URL
func URLPath(source interface{}, state data.Map) (interface{}, error) {
	resource := url.NewResource(toolbox.AsString(source))
	return resource.ParsedURL.Path, nil
}

//Hostname return host from URL
func Hostname(source interface{}, state data.Map) (interface{}, error) {
	resource := url.NewResource(toolbox.AsString(source))
	return resource.ParsedURL.Hostname(), nil
}

//AsProtobufMessage generic method for converting a map, or json string into a proto message
func AsProtobufMessage(source interface{}, state data.Map, target proto.Message) (interface{}, error) {
	var requestMap map[string]interface{}
	if toolbox.IsString(source) {
		requestMap = make(map[string]interface{})
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(toolbox.AsString(source))).Decode(&requestMap)
		if err != nil {
			fmt.Printf("failed to run udf: %v %v\n", source, err)
			return nil, err
		}
	} else {
		requestMap = toolbox.AsMap(source)
	}

	err := toolbox.DefaultConverter.AssignConverted(target, requestMap)
	if err != nil {
		fmt.Printf("failed to run udf: unable convert: %v %v\n", source, err)
		return nil, err
	}

	protodata, err := proto.Marshal(target)
	if err != nil {
		fmt.Printf("failed to run udf: unable Marshal %v %v\n", source, err)
		return nil, fmt.Errorf("failed to encode: %v, %v", requestMap, err)
	}
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	encoder.Write(protodata)
	err = encoder.Close()
	return fmt.Sprintf("base64:%v", string(buf.Bytes())), err
}

//FromProtobufMessage generic method for converting a proto message into a map
func FromProtobufMessage(source interface{}, state data.Map, sourceMessage proto.Message) (interface{}, error) {
	if toolbox.IsString(source) {
		textSource := toolbox.AsString(source)

		payload, err := util.FromPayload(textSource)
		if err != nil {
			return nil, err
		}
		err = proto.Unmarshal(payload, sourceMessage)
		if err != nil {
			return nil, err
		}

		var resultMap = make(map[string]interface{})
		err = toolbox.DefaultConverter.AssignConverted(&resultMap, sourceMessage)
		if err != nil {
			return nil, err
		}
		return toolbox.DereferenceValues(resultMap), nil
	}
	return nil, fmt.Errorf("expected string but had:%T", source)
}

// UDFs to provide a CopyHandler that performs compression before copy source to destination
// Compatible only with Object that is a content and not a directory
func CopyWithCompression(source interface{}, state data.Map) (interface{}, error) {
	// Get UDFs to Zip from context
	if zipUdf, has := getUdfFromContext("Zip", state); has {
		// Build copy handler
		var copyHandlerWithCompression storage.CopyHandler
		copyHandlerWithCompression = func(sourceObject storage.Object, reader io.Reader, destinationService storage.Service, destinationURL string) error {
			// Zip source contents
			contents, err := ioutil.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("error when reading object content before zipping source %v: %v", sourceObject.URL(), err)
			}
			zippedContents, err := zipUdf(contents, nil)
			if err != nil {
				return fmt.Errorf("error during zipping source %v: %v", sourceObject.URL(), err)
			}

			//Upload zipped contents
			if err := destinationService.Upload(destinationURL, bytes.NewReader(zippedContents.([]byte))); err != nil {
				return fmt.Errorf("error during upload, %v %v %v", sourceObject.URL(), destinationURL, err)
			}
			return nil
		}
		return copyHandlerWithCompression, nil
	}
	return nil, errors.New("unable to find udf with name Zip")
}

// UDFs to provide a CopyHandler that performs compression and corruption before copy source to destination
// Compatible only with Object that is a content and not a directory
func CopyWithCompressionAndCorruption(source interface{}, state data.Map) (interface{}, error) {
	// Get UDFs to Zip from context
	if zipUdf, has := getUdfFromContext("Zip", state); has {
		// Build copy handler
		var copyHandlerWithCompressionAndCorruption storage.CopyHandler
		copyHandlerWithCompressionAndCorruption = func(sourceObject storage.Object, reader io.Reader, destinationService storage.Service, destinationURL string) error {
			// Zip source contents
			contents, err := ioutil.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("error when reading object content before zipping source %v: %v", sourceObject.URL(), err)
			}

			//adding few bytes to corrupt the file
			contents = append(contents, '*')

			zippedContents, err := zipUdf(contents, nil)
			if err != nil {
				return fmt.Errorf("error during zipping source %v: %v", sourceObject.URL(), err)
			}

			//Upload zipped contents
			if err := destinationService.Upload(destinationURL, bytes.NewReader(zippedContents.([]byte))); err != nil {
				return fmt.Errorf("error during upload, %v %v %v", sourceObject.URL(), destinationURL, err)
			}
			return nil
		}
		return copyHandlerWithCompressionAndCorruption, nil
	}
	return nil, errors.New("unable to find udf with name Zip")
}

//RegisterProviders register the supplied providers
func RegisterProviders(providers []*endly.UdfProvider) error {
	if len(providers) == 0 {
		return nil
	}

	for _, meta := range providers {
		provider, ok := endly.UdfRegistryProvider[meta.Provider]
		if !ok {
			var available = toolbox.MapKeysToStringSlice(endly.UdfRegistryProvider)
			return fmt.Errorf("failed to lookup udf provider: %v, available: %v", meta.Provider, strings.Join(available, ","))
		}
		udf, err := provider(meta.Params...)
		if err != nil {
			return fmt.Errorf("failed to get udf from provider %v %v", meta.Provider, err)
		}
		endly.UdfRegistry[meta.ID] = udf
	}
	return nil
}
