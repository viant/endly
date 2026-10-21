package dbunit

import (
	"bytes"
	"encoding/json"
	"fmt"
	core "github.com/viant/dbunit"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"reflect"
	"strings"
)

func (s *service) hydrate(ctx *endly.Context, req *HydrateRequest) (*core.HydrateResult, error) {
	destinations, err := hydrationDestinations(req.Capture, req.Database, req.HydrateRequest)
	if err != nil {
		return nil, err
	}
	e, op, done, err := engine(ctx, req.Database, req.Assets)
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.HydrateRequest
	spec.State = snapshot(ctx, spec.State)
	spec.Tables = append([]core.HydrationTable(nil), spec.Tables...)
	for i := range spec.Tables {
		item := &spec.Tables[i]
		item.Table = expandTable(spec.State, item.Table)
		item.Source.Path = spec.State.ExpandAsText(item.Source.Path)
		item.Source.URL = spec.State.ExpandAsText(item.Source.URL)
	}
	result, err := e.Hydrate(op, spec)
	if err != nil {
		return nil, err
	}
	if err = publishHydration(ctx, destinations, result); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *service) sequence(ctx *endly.Context, req *SequenceRequest) (*core.SequenceResult, error) {
	e, op, done, err := engine(ctx, req.Database, Assets{})
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.SequenceRequest
	spec.State = snapshot(ctx, spec.State)
	result, err := e.Sequence(op, spec)
	if err == nil {
		capture(ctx, req.Capture, req.Database, workflowValue(result))
	}
	return result, err
}
func (s *service) mapFixture(ctx *endly.Context, req *MapRequest) (*core.MapResult, error) {
	e, op, done, err := engine(ctx, req.Database, req.Assets)
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.MapRequest
	spec.State = snapshot(ctx, spec.State)
	if req.SourceFrom != "" {
		if !reflect.ValueOf(spec.Source).IsZero() {
			return nil, fmt.Errorf("sourceFrom and source are mutually exclusive")
		}
		value, err := stateValue(spec.State, req.SourceFrom)
		if err != nil {
			return nil, err
		}
		if err = toolbox.DefaultConverter.AssignConverted(&spec.Source, value); err != nil {
			return nil, fmt.Errorf("sourceFrom %q: %w", req.SourceFrom, err)
		}
	}
	spec.Source.Path = spec.State.ExpandAsText(spec.Source.Path)
	spec.Source.URL = spec.State.ExpandAsText(spec.Source.URL)
	result, err := e.Map(op, spec)
	if err == nil {
		capture(ctx, req.Capture, req.Database, workflowValue(result))
	}
	return result, err
}

// decodeMappingRequest retains explicit-null defaults and core Identifier decoding
// when Endly's generic converter would otherwise bypass JSON presence hooks.
func decodeMappingRequest(_ *endly.Context, raw map[string]any) error {
	for key, value := range raw {
		if !strings.EqualFold(key, "mapping") {
			continue
		}
		content, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("mapping contract: %w", err)
		}
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.UseNumber()
		var mapping core.Mapping
		if err = decoder.Decode(&mapping); err != nil {
			return fmt.Errorf("mapping contract: %w", err)
		}
		raw[key] = mapping
		break
	}
	return nil
}
