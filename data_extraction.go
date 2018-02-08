package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/toolbox/data"
	"regexp"
	"strings"
)

//DataExtraction represents a data extraction
type DataExtraction struct {
	RegExpr string `description:"regular expression with oval bracket to extract match pattern" example:"go(\d\.\d)"`                                                                            //regular expression
	Key     string `description:"state key to store a match"`                                               //state key to store a match
	Reset   bool   `description:"reset the key in the context before evaluating this data extraction rule"` //reset the key in the context before evaluating this data extraction rule
}

//DataExtractions a slice of DataExtractions
type DataExtractions []*DataExtraction

//Extract extract data from provided inputs, the result is placed to extracted map, or error
func (d *DataExtractions) Extract(context *Context, extracted map[string]string, input ...string) error {
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
func (d *DataExtractions) Reset(state data.Map) {
	for _, extract := range *d {
		if extract.Reset {
			delete(state, extract.Key)
		}
	}
}

func matchExpression(compiledExpression *regexp.Regexp, line string, extract *DataExtraction, context *Context, extracted map[string]string) bool {
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

//NewDataExtractions creates a new NewDataExtractions
func NewDataExtractions() DataExtractions {
	return make([]*DataExtraction, 0)
}
