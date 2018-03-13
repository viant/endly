package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"regexp"
	"strings"
)

//SliceKey represents slice key
const SliceKey = "data"

//Repeater represent repeated execution
type Repeater struct {
	Extraction  Extracts  //textual regexp based data extraction
	Variables   Variables //structure data based data extraction
	Repeat      int       //how many time send this request
	SleepTimeMs int       //Sleep time after request send, this only makes sense with repeat option
	Exit        string    //Exit criteria, it uses extracted variable to determine repeat termination
}

//Extracts a slice of Extracts
type Extracts []*Extract

//Extract represents a data extraction
type Extract struct {
	RegExpr string `description:"regular expression with oval bracket to extract match pattern" example:"go(\d\.\d)"` //regular expression
	Key     string `description:"state key to store a match"`                                                         //state key to store a match
	Reset   bool   `description:"reset the key in the context before evaluating this data extraction rule"`           //reset the key in the context before evaluating this data extraction rule
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
func (r *Repeater) EvaluateExitCriteria(callerInfo string, context *Context, extracted map[string]interface{}) (bool, error) {
	var extractedState = context.state.Clone()
	for k, v := range extracted {
		extractedState[k] = v
	}
	canBreak, err := Evaluate(context, extractedState, r.Exit, callerInfo, false)
	if err != nil {
		return true, fmt.Errorf("failed to check %v exit criteia: %v", callerInfo, err)
	}
	if canBreak {
		return true, nil
	}
	return false, nil

}

func (r *Repeater) runOnce(service *AbstractService, callerInfo string, context *Context, handler func() (interface{}, error), extracted map[string]interface{}) (bool, error) {
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

	if r.Exit != "" {
		context.Publish(NewExtractEvent(extractableOutput, structuredOutput, extracted))
		if shouldBreak, err := r.EvaluateExitCriteria(callerInfo+"ExitEvaluation", context, extracted); shouldBreak || err != nil {
			return !shouldBreak, err
		}
	}
	return true, nil
}

//Run repeats x times supplied handler
func (r *Repeater) Run(service *AbstractService, callerInfo string, context *Context, handler func() (interface{}, error), extracted map[string]interface{}) error {
	for i := 0; i < r.Repeat; i++ {
		shouldContinue, err := r.runOnce(service, callerInfo, context, handler, extracted)
		if err != nil || !shouldContinue {
			return err
		}
	}
	return nil
}

//NewRepeatable creates a new repeatable struct
func NewRepeatable() *Repeater {
	return &Repeater{
		Repeat: 1,
	}
}

//Get returns non empty instance of default instance
func (r *Repeater) Get() *Repeater {
	var result = r
	if r == nil {
		result = NewRepeatable()
	}
	if result.Repeat == 0 {
		result.Repeat = 1
	}
	return result
}

//ExtractEvent  represents data extraction event
type ExtractEvent struct {
	Output           string
	StructuredOutput interface{}
	Data             interface{}
}

//NewExtractEvent creates a new event.
func NewExtractEvent(output string, structuredOutput, extracted interface{}) *ExtractEvent {
	return &ExtractEvent{
		Output:           output,
		StructuredOutput: structuredOutput,
		Data:             extracted,
	}
}

//NewExtract creates a new data extraction
func NewExtract(key, regExpr string, reset bool) *Extract {
	return &Extract{
		RegExpr: regExpr,
		Key:     key,
		Reset:   reset,
	}
}

//Extracts extract data from provided inputs, the result is placed to extracted map, or error
func (d *Extracts) Extract(context *Context, extracted map[string]interface{}, input ...string) error {
	if len(*d) == 0 || len(input) == 0 {
		return nil
	}
	for _, extract := range *d {
		if extract.Reset {
			delete(extracted, extract.Key)
		}
	}
	for _, extract := range *d {
		compiledExpression, err := regexp.Compile(extract.RegExpr)
		if err != nil {
			return fmt.Errorf("failed to extract data - invlid regexpr: %v,  %v", extract.RegExpr, err)
		}
		for _, line := range input {
			if len(line) == 0 {
				continue
			}
			if !matchExpression(compiledExpression, line, extract, context, extracted) {
				line = vtclean.Clean(line, false)
				matchExpression(compiledExpression, line, extract, context, extracted)
			}
		}
	}
	return nil
}

//Reset removes key from supplied state map.
func (d *Extracts) Reset(state data.Map) {
	for _, extract := range *d {
		if extract.Reset {
			delete(state, extract.Key)
		}
	}
}

func matchExpression(compiledExpression *regexp.Regexp, line string, extract *Extract, context *Context, extracted map[string]interface{}) bool {
	if compiledExpression.MatchString(line) {

		matched := compiledExpression.FindStringSubmatch(line)
		if extract.Key != "" {
			var state = context.State()
			var keyFragments = strings.Split(extract.Key, ".")
			for i, keyFragment := range keyFragments {
				if i+1 == len(keyFragments) {
					state.Put(extract.Key, matched[1])
					continue
				}
				if !state.Has(keyFragment) {
					state.Put(keyFragment, data.NewMap())
				}
				state = state.GetMap(keyFragment)

			}
		}
		extracted[extract.Key] = matched[1]
		return true
	}
	return false
}

//NewDataExtracts creates a new NewDataExtracts
func NewDataExtracts() Extracts {
	return make([]*Extract, 0)
}
