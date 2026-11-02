package webdriver

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
)

// Opt in only with a dedicated owned browser. Numeric controlled state is
// updated solely by trusted editing, without framework or application internals.
func TestDirectCDPControlledNumberEmptyLive(t *testing.T) {
	address := os.Getenv("ENDLY_DIRECT_CDP_TEST_ADDRESS")
	if address == "" {
		t.Skip("dedicated owned Chrome CDP endpoint required")
	}
	driver, err := newDirectCDPDriver(address)
	require.NoError(t, err)
	defer driver.Quit()
	page := `<!doctype html><title>Controlled number editing</title>
<input id="fraction" type="number" value="101"><input id="ratio" type="number" value="100">
<script>
window.edits={};
for(const field of document.querySelectorAll('input')){
 let controlled=field.value;window.edits[field.id]=[];
 field.addEventListener('input',event=>{
  window.edits[field.id].push({value:field.value,trusted:event.isTrusted,type:event.inputType});
  if(event.isTrusted)controlled=field.value;
  requestAnimationFrame(()=>{field.value=controlled});
 });
}
</script>`
	_, err = driver.OpenTab("data:text/html," + url.PathEscape(page))
	require.NoError(t, err)
	defer driver.Close()
	for _, id := range []string{"fraction", "ratio"} {
		t.Run(id, func(t *testing.T) {
			field, err := driver.FindElement(selenium.ByCSSSelector, "#"+id)
			require.NoError(t, err)
			require.NoError(t, fillElement(driver, field, ""))
			value, err := driver.ExecuteScript(`const id=arguments[0];return new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve({value:document.getElementById(id).value,edits:window.edits[id]}))));`, []interface{}{id})
			require.NoError(t, err)
			state := value.(map[string]interface{})
			require.Empty(t, state["value"])
			edits := state["edits"].([]interface{})
			require.Len(t, edits, 1)
			event := edits[0].(map[string]interface{})
			require.Equal(t, true, event["trusted"])
			require.Equal(t, "", event["value"])
			require.Equal(t, "deleteContentBackward", event["type"])
		})
	}
}
