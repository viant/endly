package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)

//SliceKey represents slice key
const SliceKey = "data"

//Repeatable represent repetable execution
type Repeatable struct {
	Extraction   DataExtractions //textual regexp based data extraction
	Variables    Variables       //structure data based data extraction
	Repeat       int             //how many time send this request
	SleepTimeMs  int             //Sleep time after request send, this only makes sense with repeat option
	ExitCriteria string          //Repeat exit criteria, it uses extracted variable to determine repeat termination
}

func asStructureData(source interface{}) data.Map {
	if source == nil {
		return data.Map(map[string]interface{}{})
	}
	var aMap = make(map[string]interface{})
	if toolbox.IsStruct(source) {
		converter = toolbox.NewColumnConverter(toolbox.DefaultDateLayout)
		converter.AssignConverted(&aMap, source)
	} else if toolbox.IsMap(source) {
		aMap = toolbox.AsMap(source)
	}
	return data.Map(aMap)
}

//AsExtractable returns extractable text and struct
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
				structuredOutput = asStructureData(value[0])
			}
		}
	default:
		structuredOutput = asStructureData(value)
	}
	if extractableOutput != "" {
		if toolbox.IsCompleteJSON(extractableOutput) {
			if strings.HasPrefix(strings.Trim(extractableOutput, " \r\n"), "[") {
				structuredOutput = data.NewMap()
				if aSlice, err := toolbox.JSONToSlice(extractableOutput); err == nil {
					structuredOutput.Put(SliceKey, aSlice)
				}
			} else if aMap, err := toolbox.JSONToMap(extractableOutput); err == nil {
				structuredOutput = data.Map(aMap)
			}
		}
	}
	return extractableOutput, structuredOutput
}

//EvaluateExitCriteria check is exit criteria is met.
func (r *Repeatable) EvaluateExitCriteria(callerInfo string, context *Context, extracted map[string]string) (bool, error) {
	var extractedState = context.state.Clone()
	for k, v := range extracted {
		extractedState[k] = v
	}
	canBreak, err := Evaluate(context,extractedState, r.ExitCriteria, callerInfo, false)
	if err != nil {
		return true, fmt.Errorf("failed to check %v exit criteia: %v", callerInfo, err)
	}
	if canBreak {
		return true, nil
	}
	return false, nil

}

func (r *Repeatable) runOnce(service *AbstractService, callerInfo string, context *Context, handler func() (interface{}, error), extracted map[string]string) (bool, error) {
	defer service.Sleep(context, r.SleepTimeMs)
	out, err := handler()
	if err != nil {
		return false, err
	}
	if out == nil {
		return true, nil
	}
	extractableOutput, structuredOutput := AsExtractable(out)



	if len(structuredOutput) > 0 {
		var extractedVariables = data.NewMap()
		err = r.Variables.Apply(structuredOutput, extractedVariables)
		for k, v := range extractedVariables {
			extracted[k] = toolbox.AsString(v)
		}
		if extractableOutput == "" {
			extractableOutput, _ = toolbox.AsJSONText(structuredOutput)
		}
	}
	err = r.Extraction.Extract(context, extracted, extractableOutput)
	if err != nil {
		return false, err
	}
	if extractableOutput != "" {
		extracted["value"] = extractableOutput //string output is published as $value
	}

	if r.ExitCriteria != "" {
		context.Publish(NewDataExtractionEvent(extractableOutput, structuredOutput, extracted))
		if shouldBreak, err := r.EvaluateExitCriteria(callerInfo+"ExitEvaluation", context, extracted); shouldBreak || err != nil {
			return !shouldBreak, err
		}
	}
	return true, nil
}

//Run repeats x times supplied handler
func (r *Repeatable) Run(service *AbstractService, callerInfo string, context *Context, handler func() (interface{}, error), extracted map[string]string) error {
	for i := 0; i < r.Repeat; i++ {
		shouldContinue, err := r.runOnce(service, callerInfo, context, handler, extracted)
		if err != nil || !shouldContinue {
			return err
		}
	}
	return nil
}

//NewRepeatable creates a new repeatable struct
func NewRepeatable() *Repeatable {
	return &Repeatable{
		Repeat: 1,
	}
}

//Get returns non empty instance of default instance
func (r *Repeatable) Get() *Repeatable {
	var result = r
	if r == nil {
		result = NewRepeatable()
	}
	if result.Repeat == 0 {
		result.Repeat = 1
	}
	return result
}


type DataExtractionEvent struct {
	Output string
	StructuredOutput interface{}
	Extracted interface{}
}

func NewDataExtractionEvent(output string, structuredOutput, extracted interface{}) *DataExtractionEvent{
	return &DataExtractionEvent{
		Output:output,
		StructuredOutput:structuredOutput,
		Extracted:extracted,
	}
}