package webdriver

import (
	"errors"

	"github.com/tebeka/selenium"
)

const (
	staleElementReferenceException = 10
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
