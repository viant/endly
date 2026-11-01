package redact

import (
	"fmt"
	"strings"
	"testing"
)

func TestRequestErrorRemovesRawAndEscapedValues(t *testing.T) {
	value := "SYNTHETIC_\"password\n<>&_雪"
	request := map[string]interface{}{"parameters": []interface{}{value}}
	message := fmt.Sprintf("unable to convert string to *int: %s; quoted %q", value, value)
	safe := RequestError(message, request)
	if strings.Contains(safe, "SYNTHETIC") || !strings.Contains(safe, "unable to convert string to *int") {
		t.Fatal("conversion reason lost or request value retained")
	}
}

func TestRequestErrorHandlesCyclesWithoutMutatingRequest(t *testing.T) {
	request := map[string]interface{}{"password": "SYNTHETIC_CYCLE_PASSWORD"}
	request["self"] = request
	safe := RequestError("invalid value SYNTHETIC_CYCLE_PASSWORD", request)
	if safe != "invalid value [REDACTED]" || request["password"] != "SYNTHETIC_CYCLE_PASSWORD" {
		t.Fatal("redaction retained a value or changed runtime request")
	}
}

func TestTextPreservesExistingBearerAndJWTRedaction(t *testing.T) {
	if got := Text("Bearer private-token eyJtest.payload.signature"); got != "Bearer [REDACTED] [REDACTED]" {
		t.Fatal("existing credential patterns were not redacted")
	}
}

func TestRequestErrorRedactsSharedBackingSlicesAndMapKeys(t *testing.T) {
	values := []string{"ordinary", "SYNTHETIC_TRAILING_PASSWORD"}
	// Slice order is deterministic; the longer alias must still be traversed.
	request := []interface{}{values[:1], values, map[string]interface{}{"SYNTHETIC_SECRET_KEY": true}}
	message := "values SYNTHETIC_TRAILING_PASSWORD key SYNTHETIC_SECRET_KEY"
	safe := RequestError(message, request)
	if strings.Contains(safe, "SYNTHETIC") {
		t.Fatal("aliased slice value or map key leaked")
	}
}

type privateDiagnosticFields struct {
	secret  string
	nested  map[string]interface{}
	pointer *string
}

func TestRequestErrorHandlesInaccessibleFields(t *testing.T) {
	secret := "SYNTHETIC_INACCESSIBLE_PASSWORD"
	request := privateDiagnosticFields{secret: secret, nested: map[string]interface{}{"private": "SYNTHETIC_NESTED_PASSWORD"}, pointer: &secret}
	safe := RequestError(secret+" SYNTHETIC_NESTED_PASSWORD", request)
	if strings.Contains(safe, "SYNTHETIC") {
		t.Fatal("inaccessible field retained")
	}
}

func TestRequestErrorPreservesReasonForSingleCharacterValue(t *testing.T) {
	message := "failed to convert a to Parameters due to Unable to convert type string into type *[]interface {}"
	safe := RequestError(message, map[string]interface{}{"parameters": []string{"a"}})
	if safe != "failed to convert [REDACTED] to Parameters due to Unable to convert type string into type *[]interface {}" {
		t.Fatal("short request value destroyed conversion field or types")
	}
}

func TestRequestErrorBoundsTraversal(t *testing.T) {
	var request interface{} = "SYNTHETIC_DEEP_PASSWORD"
	for i := 0; i < 70; i++ {
		request = []interface{}{request}
	}
	safe := RequestError("invalid value SYNTHETIC_DEEP_PASSWORD", request)
	if strings.Contains(safe, "SYNTHETIC") || !strings.Contains(safe, "traversal limit") {
		t.Fatal("unbounded request diagnostic was retained")
	}
}
