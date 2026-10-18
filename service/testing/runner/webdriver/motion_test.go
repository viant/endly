package webdriver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

type motionDriver struct {
	selenium.WebDriver
	checks      int
	stillMoving bool
}

func (d *motionDriver) ExecuteScript(string, []interface{}) (interface{}, error) {
	d.checks++
	return d.stillMoving || d.checks == 1, nil
}

func TestNativeClickWaitsForFiniteMotion(t *testing.T) {
	driver := &motionDriver{}
	require.NoError(t, waitForElementMotion(driver, nil, time.Second))
	require.Equal(t, 2, driver.checks)
}

func TestNativeClickMotionWaitIsBounded(t *testing.T) {
	require.Error(t, waitForElementMotion(&motionDriver{stillMoving: true}, nil, 0))
}

func TestNativeClickWaitsUntilControlIsEnabled(t *testing.T) {
	checks := 0
	err := (&service{}).waitForElementState(context.Background(), func() (bool, error) {
		checks++
		return checks > 1, nil
	}, time.Second, "disabled")
	require.NoError(t, err)
	require.Equal(t, 2, checks)
}
