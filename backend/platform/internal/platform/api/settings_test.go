package api_test

import (
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"path/filepath"
	"strings"
	"testing"
)

const settingsBase = "/api/v1/cognitive/settings/"

func settingsRead(t *testing.T, f *fixture, actor, section string) map[string]any {
	return decode(t, f.request(actor, "GET", settingsBase+section, nil, 200))
}
func settingsPut(t *testing.T, f *fixture, actor, section, revision string, values map[string]string, status int) map[string]any {
	return decode(t, f.request(actor, "PUT", settingsBase+section, map[string]any{"values": values, "expected_revision": revision}, status))
}
func settingsAudit(st app.State) int {
	n := 0
	for _, event := range st.Audit {
		if event.Action == "settings.update" {
			n++
		}
	}
	return n
}
func TestSettingsCASNoopAndABAPrevention(t *testing.T) {
	f := setup(t)
	statePath := filepath.Join(t.TempDir(), "settings-state.json")
	initialStore, err := app.Open(statePath)
	if err != nil {
		t.Fatal(err)
	}
	f.store = initialStore
	f.server.Store = initialStore
	initial := settingsRead(t, f, "demo-user", "profile")["revision"].(string)
	first := settingsPut(t, f, "demo-user", "profile", initial, map[string]string{"display_name": " First "}, 200)
	rev1 := first["revision"].(string)
	reopened, err := app.Open(statePath)
	if err != nil {
		t.Fatal(err)
	}
	f.store = reopened
	f.server.Store = reopened
	me := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/me", nil, 200))
	if me["name"] != "First" || me["actor_user_id"] != "demo-user" || me["role"] != "user" || me["tenant_id"] != "42" {
		t.Fatal("display update altered identity", me)
	}
	if first["values"].(map[string]any)["display_name"] != "First" {
		t.Fatal("name not normalized")
	}
	settingsPut(t, f, "demo-user", "profile", initial, map[string]string{"display_name": "Stale"}, 409)
	noop := settingsPut(t, f, "demo-user", "profile", rev1, map[string]string{"display_name": "First"}, 200)
	if noop["revision"] != rev1 || settingsAudit(f.store.Read()) != 1 {
		t.Fatal("no-op save changed version/audit")
	}
	second := settingsPut(t, f, "demo-user", "profile", rev1, map[string]string{"display_name": "Second"}, 200)
	third := settingsPut(t, f, "demo-user", "profile", second["revision"].(string), map[string]string{"display_name": "First"}, 200)
	if third["revision"] == rev1 {
		t.Fatal("ABA reused revision")
	}
	settingsPut(t, f, "demo-user", "profile", rev1, map[string]string{"display_name": "Lost update"}, 409)
	if settingsRead(t, f, "demo-user", "profile")["values"].(map[string]any)["display_name"] != "First" || settingsAudit(f.store.Read()) != 3 {
		t.Fatal("conflict changed state or audit")
	}
	settingsPut(t, f, "demo-user", "profile", third["revision"].(string), map[string]string{}, 200)
	restored := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/me", nil, 200))
	if restored["name"] != "张敏" {
		t.Fatal("clearing display name did not restore identity label")
	}
}
func TestSettingsValidationIsSectionScoped(t *testing.T) {
	f := setup(t)
	invalid := []struct {
		section string
		values  map[string]string
	}{
		{"profile", map[string]string{"notice": "wrong section"}},
		{"profile", map[string]string{"display_name": strings.Repeat("界", 81)}},
		{"general", map[string]string{"notice": "wrong section"}},
		{"general", map[string]string{"language": "unknown"}},
		{"general", map[string]string{"density": "unknown"}},
		{"general", map[string]string{"timezone": "Mars/Base"}},
		{"general", map[string]string{"timezone": "Local"}},
		{"public", map[string]string{"display_name": "wrong section"}},
	}
	for _, tc := range invalid {
		rev := settingsRead(t, f, "demo-admin", tc.section)["revision"].(string)
		settingsPut(t, f, "demo-admin", tc.section, rev, tc.values, 400)
	}
	f.request("demo-user", "PUT", settingsBase+"profile", map[string]any{}, 400)
	f.request("demo-user", "PUT", settingsBase+"public", map[string]any{"values": map[string]string{}}, 403)
	f.request("demo-user", "PUT", settingsBase+"models", map[string]any{"values": map[string]string{}}, 403)
	f.request("demo-admin", "PUT", settingsBase+"models", map[string]any{"values": map[string]string{}}, 503)
	f.request("demo-admin", "PUT", settingsBase+"unknown", map[string]any{"values": map[string]string{}}, 404)
	if settingsAudit(f.store.Read()) != 0 {
		t.Fatal("rejected settings generated mutation audit")
	}
}
func TestSettingsLegacyReadMigrationAndUserTenantIsolation(t *testing.T) {
	f := setup(t)
	if err := f.store.Update(func(st *app.State) error {
		st.Settings["demo-user:profile"] = []byte(`{"display_name":"Legacy"}`)
		st.Settings["demo-outsider:profile"] = []byte(`{"display_name":"Isolated account legacy"}`)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	legacy := settingsRead(t, f, "demo-user", "profile")
	if legacy["values"].(map[string]any)["display_name"] != "Legacy" {
		t.Fatal("legacy lost")
	}
	if settingsRead(t, f, "demo-outsider", "profile")["values"].(map[string]any)["display_name"] != "Isolated account legacy" {
		t.Fatal("known isolated-account legacy lost")
	}
	if len(settingsRead(t, f, "demo-admin", "profile")["values"].(map[string]any)) != 0 {
		t.Fatal("profile crossed user")
	}
	settingsPut(t, f, "demo-user", "profile", legacy["revision"].(string), map[string]string{"display_name": "Current"}, 200)
	if string(f.store.Read().Settings["demo-user:profile"]) != `{"display_name":"Legacy"}` {
		t.Fatal("legacy destructively changed")
	}
	if settingsRead(t, f, "demo-user", "profile")["values"].(map[string]any)["display_name"] != "Current" {
		t.Fatal("scoped write lost")
	}
	// Older clients may omit expected_revision; they still get a versioned response.
	raw := decode(t, f.request("demo-engineer", "PUT", settingsBase+"general", map[string]any{"values": map[string]string{"language": "zh-CN", "timezone": "Asia/Shanghai", "density": "comfortable"}}, 200))
	if raw["revision"] == "" {
		t.Fatal("legacy write omitted revision")
	}
}
