package webdriver

import "github.com/tebeka/selenium"

const (
	staleElementReferenceException = 10
)

func IsStaleElementError(err error) bool {
	if err == nil {
		return false
	}
	if sErr, ok := err.(*selenium.Error); ok {
		return sErr.LegacyCode == staleElementReferenceException
	}
	return false
}
