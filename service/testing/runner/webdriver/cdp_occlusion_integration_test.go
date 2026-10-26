package webdriver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tebeka/selenium"
)

func TestLiveDirectClickRejectsOcclusion(t *testing.T) {
	address := os.Getenv("ENDLY_CHROME_DEBUGGER_ADDRESS")
	if address == "" {
		t.Skip("ENDLY_CHROME_DEBUGGER_ADDRESS was not set")
	}
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><body><script>window.targetClicks=0;window.coverClicks=0;</script><button id="target" style="position:fixed;left:20px;top:80px" onclick="targetClicks++">Target</button><div id="cover" style="position:fixed;left:0;top:60px;width:180px;height:70px;z-index:10" onclick="coverClicks++">Cover</div><button id="uncover" style="position:fixed;left:250px;top:10px" onclick="document.querySelector('#cover').remove()">Uncover</button>`)
	}))
	defer page.Close()
	driver, err := newDirectCDPDriver(address)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Quit()
	if err = driver.Get(page.URL); err != nil {
		t.Fatal(err)
	}
	target, err := driver.FindElement(selenium.ByCSSSelector, "#target")
	if err != nil {
		t.Fatal(err)
	}
	if err = target.Click(); err == nil || !strings.Contains(err.Error(), "obscured") {
		t.Fatalf("covered click should be rejected: %v", err)
	}
	counts, err := driver.ExecuteScript("return [window.targetClicks,window.coverClicks];", nil)
	if err != nil || fmt.Sprint(counts) != "[0 0]" {
		t.Fatalf("covered click dispatched a mouse event: %v %v", counts, err)
	}
	uncover, err := driver.FindElement(selenium.ByCSSSelector, "#uncover")
	if err != nil {
		t.Fatal(err)
	}
	if err = uncover.Click(); err != nil {
		t.Fatal(err)
	}
	if err = target.Click(); err != nil {
		t.Fatal(err)
	}
	counts, err = driver.ExecuteScript("return [window.targetClicks,window.coverClicks];", nil)
	if err != nil || fmt.Sprint(counts) != "[1 0]" {
		t.Fatalf("native target click failed: %v %v", counts, err)
	}
}

// Move the intended button after its last stable hit-test. The native pointer
// may miss, but must never activate the unrelated button now at that point.
func TestLiveDirectClickDoesNotActivateDisplacedNeighbour(t *testing.T) {
	address := os.Getenv("ENDLY_CHROME_DEBUGGER_ADDRESS")
	if address == "" {
		t.Skip("ENDLY_CHROME_DEBUGGER_ADDRESS was not set")
	}
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><body>
<button id="target" style="position:fixed;left:20px;top:80px;width:120px;height:40px" onclick="targetClicks++">Target</button>
<button id="other" style="position:fixed;left:20px;top:240px;width:120px;height:40px" onclick="otherClicks++">Other</button>
<script>window.targetClicks=0;window.otherClicks=0;window.hitTests=0;window.displaced=false;
const hit=document.elementFromPoint.bind(document);
document.elementFromPoint=(x,y)=>{const found=hit(x,y);if(found?.id==='target'&&++hitTests===4){queueMicrotask(()=>{document.getElementById('target').style.top='240px';document.getElementById('other').style.top='80px';window.displaced=true;});}return found;};
</script>`)
	}))
	defer page.Close()
	driver, err := newDirectCDPDriver(address)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Quit()
	if err = driver.Get(page.URL); err != nil {
		t.Fatal(err)
	}
	target, err := driver.FindElement(selenium.ByCSSSelector, "#target")
	if err != nil {
		t.Fatal(err)
	}
	if err = target.Click(); err != nil {
		t.Fatal(err)
	}
	counts, err := driver.ExecuteScript("return [window.displaced,window.targetClicks,window.otherClicks];", nil)
	if err != nil || fmt.Sprint(counts) != "[true 1 0]" {
		t.Fatalf("displaced click affected the wrong control: %v %v", counts, err)
	}
	other, err := driver.FindElement(selenium.ByCSSSelector, "#other")
	if err != nil {
		t.Fatal(err)
	}
	if err = other.Click(); err != nil {
		t.Fatal(err)
	}
	counts, err = driver.ExecuteScript("return [window.targetClicks,window.otherClicks];", nil)
	if err != nil || fmt.Sprint(counts) != "[1 1]" {
		t.Fatalf("click guard survived its invocation: %v %v", counts, err)
	}
}
