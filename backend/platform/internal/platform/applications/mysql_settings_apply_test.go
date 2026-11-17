package applications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyMySQLSettingsCommitsSchema6(t *testing.T) {
	ctx := context.Background()
	dsn, root, _ := schema5SettingsRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLSettings(ctx, dsn)
	if err != nil || !plan.RoundTripVerified || !plan.ApplySupported || plan.SourceSchemaVersion != 5 {
		t.Fatal(plan, err)
	}
	if _, err = ApplyMySQLSettings(ctx, dsn, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong checksum accepted")
	}
	applied, err := ApplyMySQLSettings(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" || applied.ReadOnly {
		t.Fatal(applied, err)
	}
	var version int
	var after []byte
	if err = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &after); err != nil || version != 6 {
		t.Fatal("version not switched", version, err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(after, &object) != nil || object == nil {
		t.Fatal("invalid root after apply")
	}
	if _, ok := object["settings"]; ok {
		t.Fatal("settings still in root body")
	}
	if _, ok := object["setting_revisions"]; ok {
		t.Fatal("setting_revisions still in root body")
	}
	var settings int
	if err = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_settings`).Scan(&settings); err != nil || settings != plan.Settings {
		t.Fatal("settings rows missing", settings, plan, err)
	}
	runtime, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("runtime rejected schema6", err)
	}
	defer runtime.Close()
	st, err := runtime.readSQLContext(ctx)
	if err != nil || len(st.Settings) != plan.Settings || len(st.Apps) < 1 {
		t.Fatal("runtime state incomplete", err, len(st.Settings))
	}
	var before map[string]json.RawMessage
	if json.Unmarshal(root, &before) != nil {
		t.Fatal()
	}
	var expected map[string]json.RawMessage
	if err = json.Unmarshal(before["settings"], &expected); err != nil {
		t.Fatal(err)
	}
	for id, want := range expected {
		got, ok := st.Settings[id]
		if !ok {
			t.Fatal("missing setting after apply", id)
		}
		if !equalApplicationJSON(want, got) {
			t.Fatal("setting payload diverged", id)
		}
	}
	record, handled, err := runtime.ReadMySQLSettingsContext(ctx, "demo-user:profile", "")
	if err != nil || !handled || !record.Canonical || string(record.Value) == "{}" {
		t.Fatal("relational settings read failed", err, record)
	}
	if _, err = ApplyMySQLSettings(ctx, dsn, plan.SourceSHA256); err == nil {
		t.Fatal("repeat apply accepted")
	}
}

func TestOpenMySQLRejectsHandTunedSchema6(t *testing.T) {
	ctx := context.Background()
	dsn, _, _ := schema5SettingsRoot(t)
	db := preflightDB(t, dsn)
	plan, err := PlanMySQLSettings(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLSettings(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	var body []byte
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil {
		t.Fatal()
	}
	object["settings"] = json.RawMessage(`{}`)
	corrupt, _ := json.Marshal(object)
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, string(corrupt)); err != nil {
		t.Fatal(err)
	}
	if _, e := OpenMySQL(dsn); e == nil {
		t.Fatal("ambiguous schema6 root accepted")
	}
}