package webdriver

import (
	"fmt"
	"time"

	"github.com/tebeka/selenium"
)

// Native clicks use screen coordinates. A finite transition on an element's
// ancestor can move that target after ChromeDriver computes its click point.
// Wait only for relevant finite animations; unrelated spinners do not block.
const elementMotionScript = `
const metadata=new Set(['offset','computedOffset','easing','composite']);
const paintOnly=/^(color|backgroundColor|border(Top|Right|Bottom|Left)?Color|outlineColor|boxShadow|textShadow)$/;
for(let e=arguments[0];e;e=e.parentElement){
  if(typeof e.getAnimations!=='function')continue;
  for(const a of e.getAnimations()){
    if(a.playState!=='running'||!a.effect)continue;
    const timing=a.effect.getComputedTiming();
    if(!Number.isFinite(timing.endTime))continue;
    const keys=a.effect.getKeyframes().flatMap(frame=>Object.keys(frame).filter(key=>!metadata.has(key)));
    // Color/shadow transitions cannot move or obscure the click target.
    // Unknown properties remain conservative and retain the motion wait.
    if(keys.length&&keys.every(key=>paintOnly.test(key)))continue;
    return true;
  }
}
return false;`

func waitForElementMotion(driver selenium.WebDriver, element selenium.WebElement, timeout time.Duration) error {
	if direct, ok := driver.(*directCDPDriver); ok {
		if err := direct.bringToFront(); err != nil {
			return err
		}
	}
	deadline := time.Now().Add(timeout)
	for {
		moving, err := driver.ExecuteScript(elementMotionScript, []interface{}{element})
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
