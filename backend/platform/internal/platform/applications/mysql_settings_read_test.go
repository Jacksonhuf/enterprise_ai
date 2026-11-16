package applications

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLSettingsReadBypassesLocalMutexAndUncommittedWriter(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	key := "read:quote\"slash\\key"
	if _, e = s.UpdateMySQLSettingsContext(context.Background(), key, "", func(SettingsRecord) (*SettingsPatch, error) {
		return &SettingsPatch{Value: json.RawMessage(`{"density":"compact"}`), Revision: "r1"}, nil
	}); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	var before, after string
	if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,?,CAST(? AS JSON),?,?) WHERE id=1`, settingsJSONPath("settings", key), `{"density":"comfortable"}`, settingsJSONPath("setting_revisions", key), "r2"); e != nil {
		t.Fatal(e)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	type result struct {
		record  SettingsRecord
		handled bool
		err     error
	}
	done := make(chan result, 1)
	go func() {
		record, handled, e := s.ReadMySQLSettingsContext(ctx, key, "")
		done <- result{record, handled, e}
	}()
	select {
	case out := <-done:
		var value map[string]string
		json.Unmarshal(out.record.Value, &value)
		if out.err != nil || !out.handled || out.record.Revision != "r1" || value["density"] != "compact" {
			t.Fatal("inconsistent or blocked projection", out.err, out.record)
		}
	case <-ctx.Done():
		t.Fatal("configuration read waited for Store.mu or row writer")
	}
	if e = tx.Rollback(); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); e != nil || before != after {
		t.Fatal("read modified body", e)
	}
}
func TestMySQLSettingsReadLegacyNullFallbackAndFailure(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, handled, e := s.ReadMySQLSettingsContext(context.Background(), "key", "legacy"); handled || e != nil {
		t.Fatal("v1 did not fall back", e)
	}
	if e = s.Update(func(st *State) error { st.Settings["legacy"] = json.RawMessage(`{"density":"compact"}`); return nil }); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	record, handled, e := s.ReadMySQLSettingsContext(context.Background(), "key", "legacy")
	var value map[string]string
	json.Unmarshal(record.Value, &value)
	if e != nil || !handled || record.Canonical || value["density"] != "compact" {
		t.Fatal("legacy read", record, e)
	}
	record, handled, e = s.ReadMySQLSettingsContext(context.Background(), "key", "")
	if e != nil || !handled || record.Canonical || string(record.Value) != "{}" {
		t.Fatal("unauthorized legacy fallback", record, e)
	}
	db := preflightDB(t, dsn)
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,?,NULL)`, settingsJSONPath("settings", "key")); e != nil {
		t.Fatal(e)
	}
	record, handled, e = s.ReadMySQLSettingsContext(context.Background(), "key", "legacy")
	if e != nil || !handled || !record.Canonical || string(record.Value) != "null" {
		t.Fatal("canonical null lost priority", record, e)
	}
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.SETTINGS',JSON_OBJECT())`); e != nil {
		t.Fatal(e)
	}
	if _, handled, e = s.ReadMySQLSettingsContext(context.Background(), "key", "legacy"); handled || e != nil {
		t.Fatal("alias read did not fall back", e)
	}
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_REMOVE(body,'$.SETTINGS')`); e != nil {
		t.Fatal(e)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, handled, e = s.ReadMySQLSettingsContext(canceled, "key", ""); !handled || e == nil {
		t.Fatal("canceled read appeared successful")
	}
	if _, e = db.Exec(`UPDATE cognara_fg_state SET schema_version=99 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	if _, handled, e = s.ReadMySQLSettingsContext(context.Background(), "key", ""); !handled || e == nil {
		t.Fatal("unsupported version appeared successful")
	}
}
