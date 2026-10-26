package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/viant/toolbox/data"
)

type diagnosticPanicValue struct{}

func (diagnosticPanicValue) String() string { panic("state value must never be formatted") }
func (diagnosticPanicValue) MarshalJSON() ([]byte, error) {
	panic("state value must never be serialized")
}

func TestRequiredVariableDiagnosticNeverIncludesStateValues(t *testing.T) {
	const secret = "PRIVATE_CREDENTIAL_CANARY_8eacf"
	input := data.Map{OwnerURL: "https://user:password@host.invalid/workflows/publish.yaml?token=" + secret + "#" + secret, "archive": strings.Repeat("A", 26*1024*1024), "credentials": map[string]interface{}{"token": secret}, "opaque": diagnosticPanicValue{}, "accountId": "account"}
	err := NewVariable("workspaceId", "", "", true, nil, nil, nil, false).Apply(input, data.NewMap())
	if err == nil {
		t.Fatal("missing required input accepted")
	}
	message := err.Error()
	for _, want := range []string{"workspaceId", "was empty", "host.invalid/workflows/publish.yaml", "supply the required workflow parameter", "available keys:", "accountId", "archive", "credentials"} {
		if !strings.Contains(message, want) {
			t.Fatalf("diagnostic missing %q", want)
		}
	}
	for _, forbidden := range []string{secret, "user:password", "token=", "state dump", strings.Repeat("A", 200)} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("private state leaked (%d bytes)", len(message))
		}
	}
	if len(message) > 1536 {
		t.Fatalf("diagnostic unbounded: %d", len(message))
	}
}

func TestRequiredUnexpandedDiagnosticDoesNotEchoExpression(t *testing.T) {
	expression := "$UNRESOLVED_PRIVATE_EXPRESSION_CANARY"
	input := data.Map{OwnerURL: "file:///workflows/publish.yaml", "secret": "PRIVATE_VALUE_CANARY"}
	// Workflow parsing sets EmptyIfUnexpanded; the legacy NewVariable helper
	// ignores that argument, so use the real authored expression path.
	authored := VariableExpression("! workspaceId = " + expression)
	variable, err := authored.AsVariable()
	if err != nil {
		t.Fatal(err)
	}
	err = variable.Apply(input, data.NewMap())
	if err == nil {
		t.Fatal("unresolved required expression accepted")
	}
	message := err.Error()
	if !strings.Contains(message, "was not expanded") || !strings.Contains(message, "referenced input") {
		t.Fatalf("missing actionable unresolved reason")
	}
	if strings.Contains(message, expression) || strings.Contains(message, "PRIVATE_VALUE_CANARY") || strings.Contains(message, "state dump") {
		t.Fatal("private expression or state leaked")
	}
}

func TestRequiredVariableDiagnosticBoundsKeysOwnerAndName(t *testing.T) {
	input := data.Map{OwnerURL: "file:///" + strings.Repeat("x", 1900), "unsafe\nkey": "PRIVATE"}
	for i := 0; i < 10000; i++ {
		input[fmt.Sprintf("key%05d", i)] = diagnosticPanicValue{}
	}
	input[strings.Repeat("k", 10000)] = "PRIVATE"
	v := NewVariable(strings.Repeat("n", 2000), "missing", "", true, nil, nil, nil, false)
	err := v.Apply(input, data.NewMap())
	if err == nil {
		t.Fatal("missing input accepted")
	}
	text := err.Error()
	if len(text) > 1536 || strings.Contains(text, "\n") || !strings.Contains(text, "keys omitted") {
		t.Fatalf("diagnostic cap/sanitization failed (%d bytes)", len(text))
	}
	first := v.formatAvailableKeys(input)
	for i := 0; i < 10; i++ {
		if first != v.formatAvailableKeys(input) {
			t.Fatal("available key order is nondeterministic")
		}
	}
	for _, raw := range []interface{}{diagnosticPanicValue{}, strings.Repeat("x", 100000), "https://host.invalid/%zz"} {
		input[OwnerURL] = raw
		if owner := diagnosticOwner(input); owner != "(workflow unavailable)" {
			t.Fatal("unsafe owner was accepted")
		}
	}
}

func TestRequiredVariableValidAndOptionalBehaviorUnchanged(t *testing.T) {
	input := data.Map{OwnerURL: "file:///workflows/publish.yaml", "chosen": "workspace-1"}
	output := data.NewMap()
	if err := NewVariable("workspaceId", "chosen", "", true, nil, nil, nil, false).Apply(input, output); err != nil {
		t.Fatal(err)
	}
	if output.GetString("workspaceId") != "workspace-1" {
		t.Fatal("valid value changed")
	}
	if err := NewVariable("optional", "absent", "", false, nil, nil, nil, false).Apply(input, output); err != nil {
		t.Fatal(err)
	}
	if output.Get("optional") != nil {
		t.Fatal("optional missing value changed")
	}
}
