package model

import (
	"github.com/viant/toolbox/data"
	"strings"
	"github.com/viant/endly"
	"regexp"
	"fmt"
	"github.com/lunixbochs/vtclean"
)

//Extracts represents an extracted data collection
type Extracts []*Extract

//Extracts extract data from provided inputs, the result is placed to extracted map, or error
func (d *Extracts) Extract(context *endly.Context, extracted map[string]interface{}, input ...string) error {
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


//NewExtracts creates a new NewExtracts
func NewExtracts() Extracts {
	return make([]*Extract, 0)
}


//Extract represents a data extraction
type Extract struct {
	RegExpr string `description:"regular expression with oval bracket to extract match pattern" example:"go(\d\.\d)"` //regular expression
	Key     string `description:"state key to store a match"`                                                         //state key to store a match
	Reset   bool   `description:"reset the key in the context before evaluating this data extraction rule"`           //reset the key in the context before evaluating this data extraction rule
}


//NewExtract creates a new data extraction
func NewExtract(key, regExpr string, reset bool) *Extract {
	return &Extract{
		RegExpr: regExpr,
		Key:     key,
		Reset:   reset,
	}
}



//ExtractionEvent  represents data extraction event
type ExtractionEvent struct {
	Output           string
	StructuredOutput interface{}
	Data             interface{}
}

//NewExtractEvent creates a new event.
func NewExtractEvent(output string, structuredOutput, extracted interface{}) *ExtractionEvent {
	return &ExtractionEvent{
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

