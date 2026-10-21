package dbunit

import (
	"encoding/json"
	"fmt"
	core "github.com/viant/dbunit"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// snapshot copies workflow maps and slices without evaluating expressions or UDFs.
func snapshot(ctx *endly.Context, extra data.Map) data.Map {
	holder, _ := sessionFor(ctx)
	if holder != nil {
		holder.stateMu.Lock()
		defer holder.stateMu.Unlock()
	}
	result := cloneValue(reflect.ValueOf(ctx.State())).Interface().(data.Map)
	for k, v := range extra {
		result[k] = cloneValue(reflect.ValueOf(v)).Interface()
	}
	return result
}
func cloneValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(reflect.TypeOf((*any)(nil)).Elem())
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(cloneValue(value.Elem()))
		return result
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result.SetMapIndex(iter.Key(), cloneValue(iter.Value()))
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			result.Index(i).Set(cloneValue(value.Index(i)))
		}
		return result
	}
	return value
}
func capture(ctx *endly.Context, path, name string, value any) {
	if path == "" {
		path = "dbunit." + name
	}
	holder, _ := sessionFor(ctx)
	if holder != nil {
		holder.stateMu.Lock()
		defer holder.stateMu.Unlock()
	}
	state := ctx.State()
	state.SetValue(path, value)
}

var statePathPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z_][A-Za-z0-9_-]*|\[[0-9]+\])*$`)

// datasetsFrom reads only a captured container. It never expands templates or invokes state functions.
func datasetsFrom(state data.Map, path string) ([]core.Dataset, error) {
	value, err := stateContainer(state, path)
	if err != nil {
		return nil, err
	}
	if typed, ok := value.([]core.Dataset); ok {
		return append([]core.Dataset(nil), typed...), nil
	}
	var result []core.Dataset
	if err = toolbox.DefaultConverter.AssignConverted(&result, value); err != nil {
		return nil, fmt.Errorf("datasetsFrom %q: %w", path, err)
	}
	return result, nil
}
func stateValue(state data.Map, path string) (any, error) {
	if !statePathPattern.MatchString(path) {
		return nil, fmt.Errorf("invalid state container path %q; use a literal field/index path", path)
	}
	value := reflect.ValueOf(state)
	for _, part := range strings.FieldsFunc(path, func(r rune) bool { return r == '.' || r == '[' || r == ']' }) {
		for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
			if value.IsNil() {
				return nil, fmt.Errorf("state container %q is nil", path)
			}
			value = value.Elem()
		}
		if !value.IsValid() {
			return nil, fmt.Errorf("state container %q was not found", path)
		}
		switch value.Kind() {
		case reflect.Map:
			if value.Type().Key().Kind() != reflect.String {
				return nil, fmt.Errorf("state container %q requires string map keys", path)
			}
			value = value.MapIndex(reflect.ValueOf(part).Convert(value.Type().Key()))
		case reflect.Struct:
			found := false
			for i := 0; i < value.NumField(); i++ {
				field := value.Type().Field(i)
				if field.PkgPath != "" {
					continue
				}
				name := strings.Split(field.Tag.Get("json"), ",")[0]
				if field.Name == part || strings.EqualFold(field.Name, part) || (name != "" && name != "-" && name == part) {
					value = value.Field(i)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("state container %q was not found", path)
			}
		case reflect.Slice, reflect.Array:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= value.Len() {
				return nil, fmt.Errorf("state container %q has invalid index", path)
			}
			value = value.Index(index)
		default:
			return nil, fmt.Errorf("state container %q cannot select %q from %s", path, part, value.Kind())
		}
	}
	if !value.IsValid() {
		return nil, fmt.Errorf("state container %q was not found", path)
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, fmt.Errorf("state container %q is nil", path)
		}
		value = value.Elem()
	}
	return value.Interface(), nil
}
func stateContainer(state data.Map, path string) (any, error) {
	value, err := stateValue(state, path)
	if err != nil {
		return nil, err
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array {
		return nil, fmt.Errorf("state container %q must be a dataset list", path)
	}
	return value, nil
}

// workflowValue retains exact scalar types while making DTO fields addressable by toolbox state paths.
func workflowValue(value any) any { return workflowReflectValue(reflect.ValueOf(value)) }
func workflowReflectValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	if value.CanInterface() {
		if _, ok := value.Interface().(json.Marshaler); ok {
			return value.Interface()
		}
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return workflowReflectValue(value.Elem())
	case reflect.Struct:
		result := map[string]any{}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			result[name] = workflowReflectValue(value.Field(i))
		}
		if len(result) == 0 {
			return value.Interface()
		}
		return result
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		if value.Type().Key().Kind() != reflect.String {
			return value.Interface()
		}
		result := map[string]any{}
		iter := value.MapRange()
		for iter.Next() {
			result[iter.Key().String()] = workflowReflectValue(iter.Value())
		}
		return result
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return nil
		}
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return value.Interface()
		}
		result := make([]any, value.Len())
		for i := range result {
			result[i] = workflowReflectValue(value.Index(i))
		}
		return result
	default:
		return value.Interface()
	}
}

var capturePathPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z_][A-Za-z0-9_-]*)*$`)

func hydrationDestinations(output, name string, spec core.HydrateRequest) ([]string, error) {
	if output == "" {
		output = "dbunit." + name
	}
	paths := []string{}
	seen := map[string]bool{}
	for _, capture := range spec.StateCaptures {
		if !seen[capture.Path] {
			paths = append(paths, capture.Path)
			seen[capture.Path] = true
		}
	}
	for _, table := range spec.Tables {
		for _, capture := range table.Captures {
			if !seen[capture.Path] {
				paths = append(paths, capture.Path)
				seen[capture.Path] = true
			}
		}
	}
	paths = append(paths, output)
	for i, path := range paths {
		if !capturePathPattern.MatchString(path) {
			return nil, fmt.Errorf("invalid hydration capture path %q; use a literal dotted path", path)
		}
		for _, other := range paths[:i] {
			if path == other || strings.HasPrefix(path, other+".") || strings.HasPrefix(other, path+".") {
				return nil, fmt.Errorf("hydration capture paths %q and %q overlap", other, path)
			}
		}
	}
	return paths, nil
}

// publishHydration stages every destination before committing any workflow roots.
func publishHydration(ctx *endly.Context, paths []string, result *core.HydrateResult) error {
	holder, _ := sessionFor(ctx)
	if holder != nil {
		holder.stateMu.Lock()
		defer holder.stateMu.Unlock()
	}
	state := ctx.State()
	staged := cloneValue(reflect.ValueOf(state)).Interface().(data.Map)
	for i, path := range paths {
		var value any
		if i == len(paths)-1 {
			value = workflowValue(result)
		} else {
			var present bool
			value, present = captureMapValue(result.Captures, path)
			if !present {
				return fmt.Errorf("hydration capture %q was not returned", path)
			}
			value = workflowValue(value)
		}
		if err := setCapturedValue(map[string]any(staged), path, value); err != nil {
			return err
		}
	}
	for _, path := range paths {
		root := strings.Split(path, ".")[0]
		state[root] = staged[root]
	}
	return nil
}
func captureMapValue(values data.Map, path string) (any, bool) {
	var current any = values
	for _, part := range strings.Split(path, ".") {
		var next any
		var present bool
		switch object := current.(type) {
		case data.Map:
			next, present = object[part]
		case map[string]any:
			next, present = object[part]
		default:
			return nil, false
		}
		if !present {
			return nil, false
		}
		current = next
	}
	return current, true
}
func setCapturedValue(state map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	current := state
	for _, part := range parts[:len(parts)-1] {
		child, present := current[part]
		if !present || child == nil {
			next := map[string]any{}
			current[part] = next
			current = next
			continue
		}
		switch object := child.(type) {
		case map[string]any:
			current = object
		case data.Map:
			current = map[string]any(object)
		default:
			return fmt.Errorf("hydration capture %q has a non-object parent %q", path, part)
		}
	}
	current[parts[len(parts)-1]] = value
	return nil
}
