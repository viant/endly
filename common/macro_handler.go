package common

import (
	"fmt"
	"github.com/viant/toolbox"
	"io"
	"io/ioutil"
	"strings"
)

func NewHandler(context toolbox.Context) func(reader io.Reader) (io.Reader, error) {
	var evaluator toolbox.MacroEvaluator
	return func(reader io.Reader) (io.Reader, error) {
		if !context.GetInto(macroEvaluatorKey, evaluator) {
			return nil, fmt.Errorf("Failed to lookup MacroEvaluator")
		}
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		expanded, err := evaluator.Expand(context, toolbox.AsString(content))
		if err != nil {
			return nil, err
		}
		return strings.NewReader(toolbox.AsString(expanded)), nil
	}
}
