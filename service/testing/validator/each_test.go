package validator_test

import (
	"strings"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/validator"
	"github.com/viant/toolbox"
)

func TestAssertEachChecksEveryRecord(t *testing.T) {
	manager := endly.New()
	service, err := manager.Service(validator.ServiceID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := manager.NewContext(toolbox.NewContext())
	request := &validator.AssertRequest{Each: true, Name: "calls", Expect: map[string]interface{}{"scope": 42, "@length@rows": 1}, Actual: []interface{}{
		map[string]interface{}{"scope": 42, "rows": []int{1}},
		map[string]interface{}{"scope": 43, "rows": []int{1, 2}},
	}}
	result := service.Run(ctx, request)
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	checked := result.Response.(*validator.AssertResponse)
	if checked.FailedCount != 2 || checked.PassedCount != 2 {
		t.Fatalf("unexpected assertion counts: %+v", checked.Validation)
	}
	for _, failure := range checked.Failures {
		if !strings.Contains(failure.Path, "[1]") {
			t.Fatalf("missing failed record index: %s", failure.Path)
		}
	}
}

func TestAssertEachRejectsEmptyOrNonCollection(t *testing.T) {
	for _, actual := range []interface{}{nil, []interface{}{}, map[string]interface{}{}, "unresolved"} {
		manager := endly.New()
		service, err := manager.Service(validator.ServiceID)
		if err != nil {
			t.Fatal(err)
		}
		result := service.Run(manager.NewContext(toolbox.NewContext()), &validator.AssertRequest{Each: true, Actual: actual, Expect: 1})
		if result.Error == "" {
			t.Fatalf("accepted invalid collection: %#v", actual)
		}
	}
}
