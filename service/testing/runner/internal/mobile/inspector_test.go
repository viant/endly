package mobile

import (
	"strings"
	"testing"
)

func TestSummarizeHierarchyFiltersAndBoundsOutput(t *testing.T) {
	source := `<hierarchy><node class="android.widget.TextView" text="Welcome" bounds="[0,0][10,10]"/><node class="android.widget.Button" content-desc="Increment" enabled="true"/></hierarchy>`
	result, err := SummarizeHierarchy(source, "increment", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, `content-desc="Increment"`) || strings.Contains(result, "Welcome") {
		t.Fatalf("unexpected summary: %s", result)
	}
	truncated, err := SummarizeHierarchy(source, "", 1)
	if err != nil || !strings.Contains(truncated, "truncated after 1") {
		t.Fatalf("expected bounded output, got %q, err=%v", truncated, err)
	}
}
