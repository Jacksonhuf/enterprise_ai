package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLV3SnapshotRoundTripPreservesRawLayout(t *testing.T) {
	ctx := context.Background()
	s, db, dsn := stageMySQLV3Writers(t)
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_root',CAST('{"n":9007199254740993}' AS JSON)) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateOperational(func(st *State) error {
		st.Audit = append(st.Audit, Event{ID: "snap-event", TenantID: "42", OccurredAt: Now(), Action: "snapshot.test"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v3-snapshot.json")
	report, err := ExportMySQLSnapshot(ctx, dsn, path)
	if err != nil || report.SourceSchemaVersion != 3 || report.AuditEvents != 1 {
		t.Fatal("export", report, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if report.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("checksum")
	}
	var envelope snapshotV3
	if json.Unmarshal(data, &envelope) != nil || envelope.Version != 3 {
		t.Fatal("envelope")
	}
	if !json.Valid(envelope.StateRoot) || !strings.Contains(string(envelope.StateRoot), "9007199254740993") {
		t.Fatal("root extension lost", string(envelope.StateRoot))
	}
	target := testsupport.MySQLDSN(t)
	restored, err := ImportMySQLSnapshot(ctx, target, path, report.SHA256, 3)
	if err != nil || restored.SourceSchemaVersion != 3 || restored.AuditEvents != 1 {
		t.Fatal("import", restored, err)
	}
	again := filepath.Join(t.TempDir(), "re-export.json")
	report2, err := ExportMySQLSnapshot(ctx, target, again)
	if err != nil || report2.SourceSchemaVersion != 3 {
		t.Fatal("re-export", err)
	}
	first, _ := os.ReadFile(path)
	second, _ := os.ReadFile(again)
	var a, b snapshotV3
	json.Unmarshal(first, &a)
	json.Unmarshal(second, &b)
	if !equalApplicationJSON(a.StateRoot, b.StateRoot) {
		t.Fatal("state root drifted")
	}
	if len(a.Relational.Apps) != len(b.Relational.Apps) || len(a.Relational.Apps) < 1 {
		t.Fatal("application count", len(a.Relational.Apps))
	}
	if !equalApplicationJSON(a.Relational.Apps[0].Draft, b.Relational.Apps[0].Draft) {
		t.Fatal("draft raw drifted")
	}
	rdb := preflightDB(t, target)
	if opened, e := OpenMySQL(target); e == nil {
		opened.Close()
		t.Fatal("runtime opened schema3 import")
	}
	vtx, e := rdb.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		t.Fatal(e)
	}
	defer vtx.Rollback()
	if _, _, e = readMySQLApplicationV3(ctx, vtx, true); e != nil {
		t.Fatal("reader rejected import", e)
	}
	if _, err = db.Exec(`UPDATE cognara_fg_application_drafts SET revision='corrupt' WHERE tenant_id='42'`); err != nil {
		t.Fatal(err)
	}
	if _, err = ExportMySQLSnapshot(ctx, dsn, filepath.Join(t.TempDir(), "bad.json")); err == nil {
		t.Fatal("corrupt state exported")
	}
}
