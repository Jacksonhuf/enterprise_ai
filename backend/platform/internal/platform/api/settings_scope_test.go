package api

import (
	"context"
	"encoding/json"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http/httptest"
	"path/filepath"
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

// This tests tenant projection directly; it does not introduce a demo admin in tenant 43.
func TestGovernancePublicSettingsTenantProjection(t *testing.T) {
	store, err := app.Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	a := governance.Principal{TenantID: "42", UserID: "same-admin", Role: "admin"}
	b := governance.Principal{TenantID: "43", UserID: "same-admin", Role: "admin"}
	if err = store.Update(func(st *app.State) error {
		st.Settings["public"] = json.RawMessage(`{"notice":"Legacy tenant"}`)
		st.Settings[settingsStorageKey(b, "public")] = json.RawMessage(`{"notice":"Other tenant"}`)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: store}
	for _, tc := range []struct {
		p    governance.Principal
		want string
	}{{a, "Legacy tenant"}, {b, "Other tenant"}} {
		r := httptest.NewRequest("GET", "/api/v1/cognitive/governance/settings", nil)
		r.SetPathValue("section", "settings")
		r = r.WithContext(context.WithValue(r.Context(), principalKey{}, tc.p))
		w := httptest.NewRecorder()
		if err = server.governanceList(w, r); err != nil {
			t.Fatal(err)
		}
		var body struct {
			Items []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"items"`
		}
		if err = json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Items) != 1 || body.Items[0].Value != tc.want {
			t.Fatal("public list crossed tenant", w.Body.String(), err)
		}
	}
}
