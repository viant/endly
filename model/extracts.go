package model

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
	"regexp"
	"strings"
)

//Extracts represents an expected data collection
type Extracts []*Extract

//Extracts extract data from provided inputs, the result is placed to expected map, or error
func (d *Extracts) Extract(context *endly.Context, extracted map[string]interface{}, inputs ...string) error {
	if len(*d) == 0 || len(inputs) == 0 {
		return nil
	}

	for _, extract := range *d {
		if extract.Reset {
			delete(extracted, extract.Key)
		}
	}

	cleanedInputs := make([]string, 0)
	for _, line := range inputs {
		cleanedInputs = append(cleanedInputs, vtclean.Clean(line, false))
	}

	cleanMultiLines := strings.Join(cleanedInputs, "\n")
	multiLines := strings.Join(inputs, "\n")

	for _, extract := range *d {
		compiledExpression, err := regexp.Compile(extract.RegExpr)
		if err != nil {
			return fmt.Errorf("failed to extract data - invlid regexpr: %v,  %v", extract.RegExpr, err)
		}
		if !matchExpression(compiledExpression, multiLines, extract, context, extracted) {
			if matchExpression(compiledExpression, cleanMultiLines, extract, context, extracted) {
				continue
			}
		}
		matched := false
		for _, line := range inputs {
			if len(line) == 0 {
				continue
			}
			if matchExpression(compiledExpression, line, extract, context, extracted) {
				matched = true
				continue
			}
			cleanedLine := vtclean.Clean(line, false)
			if matchExpression(compiledExpression, cleanedLine, extract, context, extracted) {
				matched = true
				continue
			}
		}
		if extract.Required && !matched {
			if _, ok := extracted[extract.Key]; ok {
				// we found a value at some point, continue
				continue
			}
			return fmt.Errorf("failed to extract required data - no match found for regexpr: %v,  %v", extract.RegExpr, multiLines)
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

//NewExtracts creates a new NewExtracts
func NewExtracts() Extracts {
	return make([]*Extract, 0)
}

//Extract represents a data extraction
type Extract struct {
	RegExpr  string `description:"regular expression with oval bracket to extract match pattern"`            //regular expression
	Key      string `description:"state key to store a match"`                                               //state key to store a match
	Reset    bool   `description:"reset the key in the context before evaluating this data extraction rule"` //reset the key in the context before evaluating this data extraction rule
	Required bool   `description:"require that at least one pattern match is returned"`                      //require that at least one pattern match is returned
}

//NewExtract creates a new data extraction
func NewExtract(key, regExpr string, reset bool, required bool) *Extract {
	return &Extract{
		RegExpr:  regExpr,
		Key:      key,
		Reset:    reset,
		Required: required,
	}
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

func matchExpression(compiledExpression *regexp.Regexp, line string, extract *Extract, context *endly.Context, extracted map[string]interface{}) bool {
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
