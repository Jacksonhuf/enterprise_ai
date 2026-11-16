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
	st.Settings["demo-user:profile"] = json.RawMessage(`{"display_name":"Legacy person"}`)
	owner := governance.Principal{TenantID: "42", UserID: "demo-user"}
	otherTenant := governance.Principal{TenantID: "43", UserID: "demo-user"}
	if string(settingsValue(st, owner, "profile")) != `{"display_name":"Legacy person"}` || string(settingsValue(st, otherTenant, "profile")) != `{}` {
		t.Fatal("personal legacy owner tenant not checked")
	}
	if settingsStorageKey(a, "profile") == settingsStorageKey(b, "profile") || settingsRevision(st, a, "profile", json.RawMessage(`{}`)) == settingsRevision(st, b, "profile", json.RawMessage(`{}`)) {
		t.Fatal("personal versions crossed enterprise")
	}
}
