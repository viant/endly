package webdriver

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// A disposable page exercises exact role lookup after repeated in-place page
// updates. It deliberately retains zero-box display:contents wrappers and an
// inaccessible copy, without depending on any application's table implementation.
func TestDirectCDPPagedRoleLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	html := `<!doctype html><title>Native paged role lookup</title>
 <button id="next">Next Page</button><span role="status" id="status"></span>
 <div id="rows"></div><div aria-hidden="true"><button>Item-21</button></div>
 <script>let page=1;const root=document.querySelector('#rows');for(let i=0;i<10;i++){const row=document.createElement('div'),wrapper=document.createElement('span'),button=document.createElement('button');wrapper.style.display='contents';wrapper.append(button);row.append(wrapper);root.append(row)}function render(){document.querySelector('#status').textContent='Page '+page+' of 4';root.querySelectorAll('button').forEach((button,i)=>{const n=(page-1)*10+i+1;button.textContent='Item-'+String(n).padStart(2,'0');button.parentElement.parentElement.hidden=n>31});document.querySelector('#next').disabled=page===4}document.querySelector('#next').onclick=()=>{page++;render()};render();</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(html))
	require.NoError(t, err)
	defer driver.Close()
	findRole := func(name string) *WebElementSelector {
		selector, ok, err := pageSelector("getByRole", []string{"button", name})
		require.NoError(t, err)
		require.True(t, ok)
		by, value := WebSelector(selector).ByAndValue()
		return NewWebElementSelector(by, value)
	}
	for page := 1; page <= 4; page++ {
		name := fmt.Sprintf("Item-%02d", (page-1)*10+1)
		selector := findRole(name)
		elements, err := driver.FindElements(selector.By, selector.Value)
		require.NoError(t, err)
		require.Len(t, elements, 1, "page %d exact role lookup", page)
		shown, err := elements[0].IsDisplayed()
		require.NoError(t, err)
		require.True(t, shown)
		text, err := elements[0].Text()
		require.NoError(t, err)
		require.Equal(t, name, text)
		if page < 4 {
			nextSelector := findRole("Next Page")
			next, err := driver.FindElement(nextSelector.By, nextSelector.Value)
			require.NoError(t, err)
			require.NoError(t, next.Click())
		}
	}
}
