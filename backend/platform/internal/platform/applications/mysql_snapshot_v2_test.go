package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func sourceV2Snapshot(t *testing.T) (string, SnapshotReport) {
	t.Helper()
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	if e = s.UpdateOperational(func(st *State) error {
		st.Sessions["restore"] = Session{ID: "restore", AppID: "channel-replenishment", OwnerID: "demo-user", TenantID: "42", Title: "恢复核对", Messages: []Message{{ID: "m-restore", SessionID: "restore", Content: "keep"}}}
		st.Audit = append(st.Audit, Event{ID: strings.Repeat("😀", 128), TenantID: "42", OccurredAt: "0001-01-01t00:00:00z", Action: "restore"})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_state','keep')`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`UPDATE cognara_fg_audit SET payload=JSON_SET(payload,'$.future_audit','keep')`); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "v2.json")
	report, e := ExportMySQLSnapshot(context.Background(), dsn, path)
	if e != nil {
		t.Fatal(e)
	}
	if report.SourceSchemaVersion != 2 || report.AuditEvents != 1 || report.Sessions != 1 {
		t.Fatal("bad v2 summary", report)
	}
	return path, report
}
func TestMySQLV2SnapshotCompiledRestoreAndUnknownFields(t *testing.T) {
	binary := os.Getenv("COGNARA_TEST_IMPORT_BINARY")
	if binary == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("compiled importer required")
		}
		t.Skip("compiled importer not configured")
	}
	path, report := sourceV2Snapshot(t)
	target := testsupport.MySQLDSN(t)
	cmd := exec.Command(binary, "--input", path, "--expected-sha256", report.SHA256, "--source-schema-version", "2", "--mysql-dsn-env", "COGNARA_RESTORE_TARGET")
	cmd.Env = append(os.Environ(), "COGNARA_RESTORE_TARGET="+target)
	raw, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatal("compiled v2 restore failed")
	}
	var result SnapshotReport
	if json.Unmarshal(raw, &result) != nil || result.AuditEvents != 1 || result.SourceSchemaVersion != 2 {
		t.Fatal("restore report", string(raw))
	}
	restored, e := OpenMySQL(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	state := restored.Read()
	if len(state.Audit) != 1 || state.Audit[0].OccurredAt != "0001-01-01t00:00:00z" || state.Sessions["restore"].Title != "恢复核对" {
		t.Fatal("restore lost state")
	}
	output := filepath.Join(t.TempDir(), "again.json")
	if _, e = ExportMySQLSnapshot(context.Background(), target, output); e != nil {
		t.Fatal(e)
	}
	left, _ := os.ReadFile(path)
	right, _ := os.ReadFile(output)
	var a, b any
	json.Unmarshal(left, &a)
	json.Unmarshal(right, &b)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("v2 logical roundtrip changed data")
	}
	if _, e = ImportMySQLSnapshot(context.Background(), target, path, report.SHA256, 2); e == nil {
		t.Fatal("overwrote restored target")
	}
	orphan := testsupport.MySQLDSN(t)
	db, e := sql.Open("mysql", orphan)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(mysqlAuditDDL); e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	if e = insertMySQLAudit(context.Background(), tx, []Event{{ID: "orphan", TenantID: "42", OccurredAt: Now()}}, 0); e != nil {
		tx.Rollback()
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if _, e = ImportMySQLSnapshot(context.Background(), orphan, path, report.SHA256, 2); e == nil {
		t.Fatal("merged orphan target")
	}
	var count int
	if e = db.QueryRow("SELECT COUNT(*) FROM cognara_fg_state").Scan(&count); e != nil || count != 0 {
		t.Fatal("partial state restore", e)
	}
	if e = db.QueryRow("SELECT COUNT(*) FROM cognara_fg_audit").Scan(&count); e != nil || count != 1 {
		t.Fatal("orphan audit changed", e)
	}

}
func TestMySQLV2SnapshotConsistentDuringWrites(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 12; i++ {
			e := s.UpdateOperational(func(st *State) error {
				st.Settings["markers"] = []byte(strings.Repeat("1", i+1))
				st.Audit = append(st.Audit, Event{ID: ID("concurrent"), TenantID: "42", OccurredAt: Now()})
				return nil
			})
			if e != nil {
				errs <- e
				return
			}
		}
	}()
	for i := 0; i < 8; i++ {
		path := filepath.Join(t.TempDir(), "consistent.json")
		report, e := ExportMySQLSnapshot(context.Background(), dsn, path)
		if e != nil {
			wg.Wait()
			t.Fatal(e)
		}
		raw, _ := os.ReadFile(path)
		var envelope snapshotV2
		json.Unmarshal(raw, &envelope)
		var st State
		json.Unmarshal(envelope.State, &st)
		if report.AuditEvents != len(st.Settings["markers"]) {
			wg.Wait()
			t.Fatal("snapshot mixed transactions")
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}
func TestV2SnapshotRejectsAmbiguousOrDuplicateAudit(t *testing.T) {
	for _, raw := range []string{`{"format":"cognara-fg-snapshot","schema_version":2,"state":{"apps":{},"sessions":{}},"audit":null}`, `{"format":"cognara-fg-snapshot","schema_version":2,"state":{"audit":[{"event_id":"x"}]},"audit":[]}`} {
		if _, _, _, e := parseMySQLV2([]byte(raw)); e == nil {
			t.Fatal("invalid v2 accepted")
		}
	}
	event := Event{ID: "duplicate", TenantID: "42", OccurredAt: Now()}
	payload, _ := json.Marshal(event)
	raw, _ := json.Marshal(snapshotV2{snapshotV2Format, 2, json.RawMessage(`{"apps":{},"sessions":{}}`), []json.RawMessage{payload, payload}})
	if _, _, _, e := parseMySQLV2(raw); e == nil {
		t.Fatal("duplicate audit accepted")
	}
}
