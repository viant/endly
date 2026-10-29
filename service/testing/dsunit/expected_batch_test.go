package dsunit

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestCollectExpectedFreezesEachHydratedCase(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	state := ctx.State()
	for _, id := range []int{101, 202} {
		state.SetValue("fixtureID", id)
		_, err := collectExpected(ctx, &CollectExpectedRequest{Bucket: "group.expected", Datastore: "fixture", Data: map[string][]map[string]interface{}{"record": {{"ID": "$fixtureID", "nullable": nil, "enabled": false, "amount": 0}}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	raw, ok := state.GetValue("group.expected.fixture")
	if !ok {
		t.Fatal("batch absent")
	}
	var batch map[string][]map[string]interface{}
	if err := toolbox.DefaultConverter.AssignConverted(&batch, raw); err != nil {
		t.Fatal(err)
	}
	rows := batch["record"]
	if len(rows) != 2 || toolbox.AsInt(rows[0]["ID"]) != 101 || toolbox.AsInt(rows[1]["ID"]) != 202 {
		t.Fatalf("case identities not frozen: %#v", rows)
	}
	for _, row := range rows {
		if row["nullable"] != nil || row["enabled"] != false || toolbox.AsInt(row["amount"]) != 0 {
			t.Fatalf("changed expected values: %#v", row)
		}
	}
}
func TestCollectExpectedRejectsEmptyCase(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	if _, err := collectExpected(ctx, &CollectExpectedRequest{Bucket: "group.expected", Datastore: "fixture"}); err == nil {
		t.Fatal("empty expectation accepted")
	}
}
