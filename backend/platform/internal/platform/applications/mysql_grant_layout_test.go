package applications

import (
	"encoding/json"
	"testing"
)

func TestGrantLayoutRoundTripPreservesSeedGrants(t *testing.T) {
	_, root := schema3GrantFixture(t)
	if err := verifyGrantLayout(root); err != nil {
		t.Fatal(err)
	}
	layout, err := splitGrantLayout(root)
	if err != nil || len(layout.Grants) < 1 {
		t.Fatal(err, len(layout.Grants))
	}
	rebuilt, err := joinGrantLayout(layout)
	if err != nil || !equalApplicationJSON(root, rebuilt) {
		t.Fatal("round trip", err)
	}
}

func TestGrantLayoutRejectsAggregateAuthority(t *testing.T) {
	_, root := schema3GrantFixture(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	object["apps"] = json.RawMessage(`{}`)
	bad, _ := json.Marshal(object)
	if _, err := splitGrantLayout(bad); err == nil {
		t.Fatal("apps in root accepted")
	}
}