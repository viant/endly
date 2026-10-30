package webdriver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

func TestLiveMotionWaitDistinguishesPaintAndGeometry(t *testing.T) {
	address := os.Getenv("ENDLY_CHROME_DEBUGGER_ADDRESS")
	if address == "" {
		t.Skip("ENDLY_CHROME_DEBUGGER_ADDRESS was not set")
	}
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><button id="target">Target</button>`))
	}))
	defer page.Close()
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	require.NoError(t, driver.Get(page.URL))
	target, err := driver.FindElement(selenium.ByCSSSelector, "#target")
	require.NoError(t, err)
	_, err = driver.ExecuteScript(`window.paint=document.querySelector('#target').animate([{backgroundColor:'red',color:'white',boxShadow:'0 0 0 black'},{backgroundColor:'blue',color:'black',boxShadow:'0 0 5px black'}],3000);return true;`, nil)
	require.NoError(t, err)
	require.NoError(t, waitForElementMotion(driver, target, 200*time.Millisecond))
	state, err := driver.ExecuteScript(`return paint.playState;`, nil)
	require.NoError(t, err)
	require.Equal(t, "running", state, "paint-only animation need not finish before interaction")
	_, err = driver.ExecuteScript(`paint.cancel();window.motion=document.querySelector('#target').animate([{transform:'translateX(0)'},{transform:'translateX(50px)'}],120);return true;`, nil)
	require.NoError(t, err)
	require.NoError(t, waitForElementMotion(driver, target, time.Second))
	state, err = driver.ExecuteScript(`return motion.playState;`, nil)
	require.NoError(t, err)
	require.Equal(t, "finished", state, "geometry animation must finish before coordinate interaction")
}
