package webdriver

import (
	"errors"
	"testing"

	"github.com/tebeka/selenium"
)

func TestIsRetryableElementError(t *testing.T) {
	for _, err := range []error{
		&selenium.Error{LegacyCode: staleElementReferenceException},
		&selenium.Error{LegacyCode: elementNotInteractableException},
		errors.New("element click intercepted: other element would receive the click"),
		errors.New("javascript error: element is not editable"),
	} {
		if !IsRetryableElementError(err) {
			t.Fatalf("expected retryable error: %v", err)
		}
	}
	if IsRetryableElementError(errors.New("invalid selector")) {
		t.Fatal("invalid selector must not be retried")
	}
	if IsRetryableElementError(errors.New("fill requires a text input")) {
		t.Fatal("an unsupported control type must not be retried")
	}
}

func TestJavaScriptClickCandidateExcludesStaleElement(t *testing.T) {
	if isJavaScriptClickCandidate(&selenium.Error{LegacyCode: staleElementReferenceException}) {
		t.Fatal("stale elements must be re-resolved, not clicked with JavaScript")
	}
	if !isJavaScriptClickCandidate(errors.New("element not interactable")) {
		t.Fatal("expected interaction failure to be eligible")
	}
}

func TestUndeliveredNativeClickCanRetryButCannotUseJavaScript(t *testing.T) {
	err := errors.New("native click did not reach target")
	if !IsRetryableElementError(err) {
		t.Fatal("undelivered native click should retry within the action deadline")
	}
	if isJavaScriptClickCandidate(err) {
		t.Fatal("undelivered native clicks must not bypass disabled or refreshed controls with JavaScript")
	}
}

func TestObscuredNativeClickCanRetryButCannotUseJavaScript(t *testing.T) {
	err := errors.New("click target remained obscured or moving")
	if !IsRetryableElementError(err) {
		t.Fatal("native hit testing should retry within the action deadline")
	}
	if isJavaScriptClickCandidate(err) {
		t.Fatal("native hit testing must not bypass occlusion with JavaScript")
	}
}
