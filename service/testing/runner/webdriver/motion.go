package webdriver

import (
	"fmt"
	"time"

	"github.com/tebeka/selenium"
)

// Native clicks use screen coordinates. A finite transition on an element's
// ancestor can move that target after ChromeDriver computes its click point.
// Wait only for relevant finite animations; unrelated spinners do not block.
func waitForElementMotion(driver selenium.WebDriver, element selenium.WebElement, timeout time.Duration) error {
	if direct, ok := driver.(*directCDPDriver); ok {
		if err := direct.bringToFront(); err != nil {
			return err
		}
	}
	deadline := time.Now().Add(timeout)
	for {
		moving, err := driver.ExecuteScript(`for(let e=arguments[0];e;e=e.parentElement){if(typeof e.getAnimations!=='function')continue;for(const a of e.getAnimations()){if(a.playState!=='running'||!a.effect)continue;const timing=a.effect.getComputedTiming();if(Number.isFinite(timing.endTime))return true;}}return false;`, []interface{}{element})
		if err != nil {
			return err
		}
		if moving != true {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("element remained in motion before native click")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
