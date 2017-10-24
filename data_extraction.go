package endly

import (
	"fmt"
	"regexp"
	"strings"
	"github.com/viant/toolbox/data"
)

type DataExtraction struct {
	RegExpr string
	Key     string
}

type DataExtractions []*DataExtraction

func (d *DataExtractions) Extract(context *Context, extracted map[string]string, input ...string) error {
	if len(*d) == 0 || len(input) == 0 {
		return nil
	}
	for _, extract := range *d {
		compiledExpression, err := regexp.Compile(extract.RegExpr)
		if err != nil {
			return fmt.Errorf("Failed to extract data - invlid regexpr: %v,  %v", extract.RegExpr, err)
		}
		for _, line := range input {
			if len(line) == 0 {
				continue
			}
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

			}
		}
	}
	return nil
}

func NewDataExtractions() DataExtractions {
	return make([]*DataExtraction, 0)
}
