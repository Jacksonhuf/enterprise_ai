package applications

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLTargetedSettingsAtomicityAndOpaqueFields(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	// v1 declines before calling the mutation; the caller retains its aggregate path.
	invoked := false
	if handled, e := s.UpdateMySQLSettingsContext(context.Background(), "key", "", func(SettingsRecord) (*SettingsPatch, error) { invoked = true; return nil, nil }); handled || e != nil || invoked {
		t.Fatal("v1 did not fall back", e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	// Unrecognized nested properties inside OTHER domains must survive a targeted write.
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future',CAST(? AS JSON),'$.apps."channel-replenishment".future',CAST(? AS JSON),'$.settings',NULL,'$.setting_revisions',NULL)`, `{"n":9007199254740993}`, `{"keep":true}`); e != nil {
		t.Fatal(e)
	}
	key := "settings:quote\"slash\\key"
	patch := func(record SettingsRecord) (*SettingsPatch, error) {
		if record.Canonical || string(record.Value) != "{}" || record.Revision != "" {
			t.Fatal("unexpected initial projection", record)
		}
		return &SettingsPatch{Value: json.RawMessage(`{"density":"compact"}`), Revision: "r1", Audit: []Event{{TenantID: "42", ID: "settings-first", OccurredAt: Now()}}}, nil
	}
	if handled, e := s.UpdateMySQLSettingsContext(context.Background(), key, "", patch); !handled || e != nil {
		t.Fatal("targeted save", e)
	}
	read := func() string {
		var raw string
		if e := db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		return raw
	}
	before := read()
	if !strings.Contains(before, `9007199254740993`) || !strings.Contains(before, `"future": {"keep": true}`) {
		t.Fatal("opaque fields lost", before)
	}
	if got := s.Read(); string(got.Settings[key]) != `{"density": "compact"}` || got.SettingRevisions[key] != "r1" || len(got.Audit) != 1 {
		t.Fatal("targeted update not visible")
	}
	// No-op and failed callbacks never update the body or append an event.
	if _, e = s.UpdateMySQLSettingsContext(context.Background(), key, "", func(record SettingsRecord) (*SettingsPatch, error) {
		if !record.Canonical || record.Revision != "r1" {
			t.Fatal("missing current projection")
		}
		return nil, nil
	}); e != nil {
		t.Fatal(e)
	}
	if read() != before {
		t.Fatal("no-op rewrote state")
	}
	if _, e = s.UpdateMySQLSettingsContext(context.Background(), key, "", func(SettingsRecord) (*SettingsPatch, error) { return nil, errors.New("conflict") }); e == nil {
		t.Fatal("callback error ignored")
	}
	if read() != before {
		t.Fatal("conflict rewrote state")
	}
	// A duplicate audit insertion rolls back both configuration and revision.
	if _, e = s.UpdateMySQLSettingsContext(context.Background(), key, "", func(SettingsRecord) (*SettingsPatch, error) {
		return &SettingsPatch{Value: json.RawMessage(`{"density":"comfortable"}`), Revision: "r2", Audit: []Event{{TenantID: "42", ID: "settings-first", OccurredAt: Now()}}}, nil
	}); e == nil {
		t.Fatal("duplicate audit committed")
	}
	if read() != before || len(s.Read().Audit) != 1 {
		t.Fatal("partial settings transaction")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	invoked = false
	if _, e = s.UpdateMySQLSettingsContext(canceled, key, "", func(SettingsRecord) (*SettingsPatch, error) { invoked = true; return nil, nil }); e == nil || invoked {
		t.Fatal("canceled mutation ran")
	}
}
func TestMySQLTargetedSettingsLegacyAndAliasFallback(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Update(func(st *State) error {
		st.Settings["demo-user:general"] = json.RawMessage(`{"density":"compact"}`)
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	if handled, e := s.UpdateMySQLSettingsContext(context.Background(), "new", "demo-user:general", func(record SettingsRecord) (*SettingsPatch, error) {
		var v map[string]string
		json.Unmarshal(record.Value, &v)
		if record.Canonical || v["density"] != "compact" {
			t.Fatal("legacy value missing")
		}
		return &SettingsPatch{Value: record.Value, Revision: "r1"}, nil
	}); !handled || e != nil {
		t.Fatal(e)
	}
	if string(s.Read().Settings["new"]) == "" || s.Read().Settings["demo-user:general"] == nil {
		t.Fatal("legacy upgrade destructive")
	}
	db := preflightDB(t, dsn)
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.SETTINGS',JSON_OBJECT())`); e != nil {
		t.Fatal(e)
	}
	invoked := false
	if handled, e := s.UpdateMySQLSettingsContext(context.Background(), "new", "", func(SettingsRecord) (*SettingsPatch, error) { invoked = true; return nil, nil }); handled || e != nil || invoked {
		t.Fatal("alias envelope did not fall back", e)
	}
}
