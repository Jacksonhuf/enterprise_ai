package api_test

import (
	"context"
	"testing"

	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLSettingsAndIdentityReadsFailExplicitlyWhenUnavailable(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	store, e := app.OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if e = app.MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	f := setup(t)
	f.store = store
	f.server.Store = store
	before := settingsRead(t, f, "demo-user", "profile")
	saved := settingsPut(t, f, "demo-user", "profile", before["revision"].(string), map[string]string{"display_name": "当前资料"}, 200)
	if settingsRead(t, f, "demo-user", "profile")["revision"] != saved["revision"] {
		t.Fatal("projected revision differs from save")
	}
	me := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/me", nil, 200))
	if me["name"] != "当前资料" || me["actor_user_id"] != "demo-user" || me["role"] != "user" {
		t.Fatal("projected profile changed identity")
	}
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{settingsBase + "profile", settingsBase + "general", "/api/v1/cognitive/me"} {
		out := decode(t, f.request("demo-user", "GET", path, nil, 503))
		if out["values"] != nil || out["revision"] != nil || out["name"] != nil {
			t.Fatal("failure returned defaults as success", out)
		}
		envelope, ok := out["error"].(map[string]any)
		if !ok || envelope["code"] != "CAPABILITY_UNAVAILABLE" {
			t.Fatal("wrong error envelope", out)
		}
	}
	f.request("demo-admin", "GET", "/api/v1/cognitive/governance/settings", nil, 503)
}
