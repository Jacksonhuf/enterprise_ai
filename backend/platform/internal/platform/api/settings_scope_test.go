package api

import (
	"encoding/json"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"testing"
)

func TestSettingsNamespaceSeparatesEnterprisePublicConfiguration(t *testing.T) {
	a := governance.Principal{TenantID: "42", UserID: "same-user"}
	b := governance.Principal{TenantID: "43", UserID: "same-user"}
	st := app.State{Settings: map[string]json.RawMessage{"public": json.RawMessage(`{"notice":"Legacy tenant"}`)}}
	if string(settingsValue(st, a, "public")) != `{"notice":"Legacy tenant"}` || string(settingsValue(st, b, "public")) != `{}` {
		t.Fatal("legacy public scope leaked")
	}
	st.Settings[settingsStorageKey(b, "public")] = json.RawMessage(`{"notice":"Other enterprise"}`)
	if string(settingsValue(st, a, "public")) == string(settingsValue(st, b, "public")) {
		t.Fatal("enterprise namespaces collided")
	}
	if settingsStorageKey(a, "profile") == settingsStorageKey(b, "profile") || settingsRevision(st, a, "profile", json.RawMessage(`{}`)) == settingsRevision(st, b, "profile", json.RawMessage(`{}`)) {
		t.Fatal("personal versions crossed enterprise")
	}
}
