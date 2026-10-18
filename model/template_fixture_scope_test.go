package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplate_FixtureTagsMatchWindowScope(t *testing.T) {
	dir := t.TempDir()
	for _, group := range []string{"advertiser", "line"} {
		folder := filepath.Join(dir, group, "001_properties_read", "dbsetup")
		if err := os.MkdirAll(folder, 0700); err != nil {
			t.Fatal(err)
		}
		data := `[{"ID":"$Sequences.CI_ACCOUNT/${tag}.owner","NAME":"${tag}"}]`
		if err := os.WriteFile(filepath.Join(folder, "CI_ACCOUNT.json"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source := `pipeline:
  advertiser:
    tag: advertiser_$pathMatch
    subPath: advertiser/${index}_*
    data:
      '[]ci_ads_dbsetup': '@dbsetup'
    template:
      run:
        action: nop
        init: {tag: '$tag'}
  line:
    tag: line_$pathMatch
    subPath: line/${index}_*
    data:
      '[]ci_ads_dbsetup': '@dbsetup'
    template:
      run:
        action: nop
        init: {tag: '$tag'}
`
	filename := filepath.Join(dir, "suite.yaml")
	if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	inline, err := loadInlineWorkflow(filename)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := inline.AsWorkflow("suite", dir)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(workflow.Data)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, group := range []string{"advertiser", "line"} {
		expected := "$Sequences.CI_ACCOUNT/" + group + "_properties_read.owner"
		if !strings.Contains(text, expected) {
			t.Fatalf("missing isolated fixture alias %q in %s", expected, text)
		}
	}
	if strings.Contains(text, "CI_ACCOUNT/properties_read.owner") {
		t.Fatalf("unscoped fixture alias remains: %s", text)
	}
}

// Literal display tags historically leave fixture tags based on the folder.
// Keep that behavior, and keep the default path tag unchanged.
func TestTemplate_FixtureTagCompatibility(t *testing.T) {
	for _, tag := range []string{"", "$pathMatch", "literal"} {
		t.Run(tag, func(t *testing.T) {
			dir := t.TempDir()
			folder := filepath.Join(dir, "cases", "001_properties_read", "dbsetup")
			if err := os.MkdirAll(folder, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(folder, "CI_ACCOUNT.json"), []byte(`[{"ID":"$Sequences.CI_ACCOUNT/${tag}.owner"}]`), 0600); err != nil {
				t.Fatal(err)
			}
			source := "pipeline:\n  cases:\n    tag: '" + tag + "'\n    subPath: cases/${index}_*\n    data:\n      '[]ci_ads_dbsetup': '@dbsetup'\n    template:\n      run:\n        action: nop\n        init: {tag: '$tag'}\n"
			filename := filepath.Join(dir, "suite.yaml")
			if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			inline, err := loadInlineWorkflow(filename)
			if err != nil {
				t.Fatal(err)
			}
			workflow, err := inline.AsWorkflow("suite", dir)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(workflow.Data)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(encoded), "$Sequences.CI_ACCOUNT/properties_read.owner") {
				t.Fatalf("fixture compatibility changed: %s", encoded)
			}
		})
	}
}
