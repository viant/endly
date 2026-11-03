package workflow_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	_ "github.com/viant/endly/service/system/storage"
	_ "github.com/viant/endly/service/testing/validator"
	"github.com/viant/endly/service/workflow"
	"github.com/viant/toolbox/data"
)

type stateJSONProduceRequest struct{}

func TestNativeStateJSONBytesRetainsCompletedResponse(t *testing.T) {
	manager := endly.New()
	payload := map[string]any{
		"code":    `const a = "${binding}"; const b = "$known"; return "$Touched()";`,
		"unknown": `${unknown} $missing`, "utf8": "你好 café 🚀",
		"zero": 0, "false": false, "null": nil,
		"number": json.Number("9007199254740993"), "numericText": "9007199254740993",
		"emptyList": []any{}, "emptyMap": map[string]any{},
		"nested": []any{map[string]any{"template": "${binding}", "flag": false}},
	}
	producer := endly.NewAbstractService("state-json-fixture")
	producer.Register(&endly.Route{Action: "produce", RequestProvider: func() any { return &stateJSONProduceRequest{} }, ResponseProvider: func() any { return &map[string]any{} }, Handler: func(*endly.Context, any) (any, error) { return payload, nil }})
	manager.Register(producer)
	ctx := manager.NewContext(nil)
	defer ctx.Close()
	state := ctx.State()
	state.Put("binding", "must-not-enter-data")
	state.Put("known", "must-not-enter-data")
	calls := 0
	state.Put("Touched", func(any, data.Map) (any, error) { calls++; return "must-not-execute", nil })
	path := filepath.Join(t.TempDir(), "response.json")
	var response any
	request := &workflow.RunRequest{Name: "state-json", AssetURL: filepath.Join(t.TempDir(), "workflow.yaml"), SharedState: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{
		{Key: "produced", Value: map[string]any{"action": "state-json-fixture:produce"}},
		{Key: "save", Value: map[string]any{"action": "storage:upload", "init": []string{"payloadBytes = $StateJSONBytes('produced')"}, "sourceKey": "payloadBytes", "dest": map[string]any{"URL": path}}},
	}}}
	require.NoError(t, endly.Run(ctx, request, &response))
	require.IsType(t, []byte{}, state["payloadBytes"], "the native variable boundary must retain binary output")
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	expected, err := json.MarshalIndent(payload, "", "\t")
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Zero(t, calls)
	decoder := json.NewDecoder(bytes.NewReader(actual))
	decoder.UseNumber()
	var result map[string]any
	require.NoError(t, decoder.Decode(&result))
	require.Equal(t, json.Number("9007199254740993"), result["number"])
	require.Equal(t, "9007199254740993", result["numericText"])
	require.Equal(t, false, result["false"])
	require.Equal(t, json.Number("0"), result["zero"])
	require.Nil(t, result["null"])
	require.Equal(t, expected, []byte(string(state["payloadBytes"].([]byte))))
}

func TestNativeUploadTemplateAndBinaryCompatibility(t *testing.T) {
	for _, mode := range []string{"loaded-text-template", "authored-text-template", "existing-binary", "decoded-binary"} {
		t.Run(mode, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			defer ctx.Close()
			state := ctx.State()
			state.Put("binding", "expanded")
			literal := []byte(`literal ${binding} $unknown`)
			state.Put("loaded", "loaded ${binding}")
			state.Put("binary", literal)
			state.Put("encoded", base64.StdEncoding.EncodeToString(literal))
			path := filepath.Join(t.TempDir(), "asset.txt")
			step := map[string]any{"action": "storage:upload", "sourceKey": "payload", "dest": map[string]any{"URL": path}}
			want := string(literal)
			switch mode {
			case "loaded-text-template":
				step["sourceKey"] = "loaded"
				want = "loaded expanded"
			case "authored-text-template":
				step["init"] = []string{"payload = authored ${binding}"}
				want = "authored expanded"
			case "existing-binary":
				step["sourceKey"] = "binary"
			case "decoded-binary":
				step["init"] = []string{"payload = $Base64Decode($encoded)"}
			}
			var out any
			require.NoError(t, endly.Run(ctx, &workflow.RunRequest{Name: "compatibility", AssetURL: filepath.Join(t.TempDir(), "workflow.yaml"), SharedState: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "save", Value: step}}}}, &out))
			actual, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, want, string(actual))
			if mode == "decoded-binary" {
				require.IsType(t, []byte{}, state["payload"])
			}
		})
	}
}

type unsafeJSONFixture struct{}

func (unsafeJSONFixture) MarshalJSON() ([]byte, error) {
	return nil, errors.New("private-payload-canary must not be disclosed")
}

func TestNativeStateJSONBytesFailureIsBoundedAndRedacted(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	state := ctx.State()
	state.Put("unsupported", unsafeJSONFixture{})
	var captured bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&captured)
	defer log.SetOutput(previous)
	path := filepath.Join(t.TempDir(), "must-not-exist.json")
	request := &workflow.RunRequest{Name: "invalid", AssetURL: filepath.Join(t.TempDir(), "workflow.yaml"), SharedState: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{{Key: "save", Value: map[string]any{
		"action": "storage:upload", "init": []any{map[string]any{"name": "payload", "value": "$StateJSONBytes(unsupported)", "required": true, "emptyIfUnexpanded": true}}, "sourceKey": "payload", "dest": map[string]any{"URL": path},
	}}}}}
	var out any
	err := endly.Run(ctx, request, &out)
	require.Error(t, err)
	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr), "failed serialization must not produce a receipt")
	require.NotContains(t, captured.String(), "private-payload-canary")
	require.NotContains(t, err.Error(), "private-payload-canary")
	require.NotContains(t, strings.ToLower(captured.String()), "marshaljson")
}

func TestNativeStateJSONBytesValidatorComparison(t *testing.T) {
	for _, conversion := range []string{"AsString", "Base64Encode", "prefixed-AsString"} {
		t.Run(conversion, func(t *testing.T) {
			manager := endly.New()
			payload := map[string]any{"code": "${binding} $known $Touched() ${unknown} $missing", "utf8": "你好", "zero": 0, "flag": false, "null": nil}
			producer := endly.NewAbstractService("state-json-compare-fixture")
			producer.Register(&endly.Route{Action: "produce", RequestProvider: func() any { return &stateJSONProduceRequest{} }, ResponseProvider: func() any { return &map[string]any{} }, Handler: func(*endly.Context, any) (any, error) { return payload, nil }})
			manager.Register(producer)
			ctx := manager.NewContext(nil)
			defer ctx.Close()
			state := ctx.State()
			state.Put("binding", "expanded")
			state.Put("known", "expanded")
			calls := 0
			state.Put("Touched", func(any, data.Map) (any, error) { calls++; return "executed", nil })
			expected, err := json.MarshalIndent(payload, "", "\t")
			require.NoError(t, err)
			udf := conversion
			actual, expect := "actualText", "$expectedText"
			if conversion == "prefixed-AsString" {
				udf = "AsString"
				actual = "definition:${actualText}"
				expect = "definition:${expectedText}"
			}

			var out any
			request := &workflow.RunRequest{Name: "compare", AssetURL: filepath.Join(t.TempDir(), "workflow.yaml"), SharedState: true, Inlined: &model.Inlined{Pipeline: []*model.MapEntry{
				{Key: "produced", Value: map[string]any{"action": "state-json-compare-fixture:produce"}},
				{Key: "reference", Value: map[string]any{"action": "state-json-compare-fixture:produce"}},
				{Key: "comparison", Value: map[string]any{"action": "validator:assert", "init": []string{"comparisonBytes = $StateJSONBytes('produced')", "referenceBytes = $StateJSONBytes('reference')", "actualText = $" + udf + "($comparisonBytes)", "expectedText = $" + udf + "($referenceBytes)"}, "actual": actual, "expect": expect}},
				{Key: "different", Value: map[string]any{"action": "validator:assert", "actual": "actualText", "expect": "different"}},
			}}}
			require.NoError(t, endly.Run(ctx, request, &out))
			want := string(expected)
			if conversion == "Base64Encode" {
				want = base64.StdEncoding.EncodeToString(expected)
			}
			require.Equal(t, want, state["actualText"])
			require.Equal(t, want, state["expectedText"])
			require.Zero(t, calls)
			comparison := state["comparison"].(map[string]any)
			require.EqualValues(t, 0, comparison["FailedCount"])
			require.NotEqualValues(t, 0, comparison["PassedCount"])
			if conversion == "prefixed-AsString" {
				require.EqualValues(t, 1, comparison["PassedCount"])
			}
			different := state["different"].(map[string]any)
			require.NotEqualValues(t, 0, different["FailedCount"])
		})
	}
}
