package endly

import (
	"fmt"
	"github.com/viant/endly/common"
	"regexp"
	"strings"
)

type DataExtraction struct {
	Name     string
	RegExpr  string
	StateKey string
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
				return nil
			}
			if compiledExpression.MatchString(line) {

				matched := compiledExpression.FindStringSubmatch(line)
				if extract.StateKey != "" {
					var state = context.State()
					var keyFragments = strings.Split(extract.StateKey, ".")
					for i, keyFragment := range keyFragments {
						if i+1 == len(keyFragments) {
							state.Put(extract.StateKey, matched[1])
							continue
						}

						if !state.Has(keyFragment) {
							state.Put(keyFragment, common.NewMap())
						}
						state = state.GetMap(keyFragment)

					}
					if extract.Name == "" {
						extract.Name = extract.StateKey
					}

				}
				extracted[extract.Name] = matched[1]

			}
		}
	}
	return nil
}

func NewDataExtractions() DataExtractions {
	return make([]*DataExtraction, 0)
}
