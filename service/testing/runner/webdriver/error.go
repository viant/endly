package webdriver

import (
	"errors"
	"strings"

	"github.com/tebeka/selenium"
)

const (
	staleElementReferenceException   = 10
	elementNotVisibleException       = 11
	elementNotInteractableException  = 60
	elementClickInterceptedException = 64
)

func IsStaleElementError(err error) bool {
	if err == nil {
		return false
	}
	var sErr *selenium.Error
	if errors.As(err, &sErr) {
		return sErr.LegacyCode == staleElementReferenceException
	}
	return false
}

// IsRetryableElementError identifies transient DOM/layout failures that can
// disappear after a framework render, animation, or overlay transition.
func IsRetryableElementError(err error) bool {
	if err == nil {
		return false
	}
	var sErr *selenium.Error
	if errors.As(err, &sErr) {
		switch sErr.LegacyCode {
		case staleElementReferenceException, elementNotVisibleException, elementNotInteractableException, elementClickInterceptedException:
			return true
		}
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"stale element",
		"element is not attached",
		"element not interactable",
		"element is not displayed",
		"element click intercepted",
		"not clickable at point",
		"other element would receive",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func isJavaScriptClickCandidate(err error) bool {
	if err == nil {
		return false
	}
	var sErr *selenium.Error
	if errors.As(err, &sErr) {
		switch sErr.LegacyCode {
		case elementNotVisibleException, elementNotInteractableException, elementClickInterceptedException:
			return true
		}
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not interactable") ||
		strings.Contains(message, "click intercepted") ||
		strings.Contains(message, "not clickable at point") ||
		strings.Contains(message, "other element would receive")
}
