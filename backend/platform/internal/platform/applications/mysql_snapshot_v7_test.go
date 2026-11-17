package applications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func schema7ExceptionSnapshotFixture(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	dsn, _ := schema6ExceptionRoot(t)
	plan, err := PlanMySQLExceptions(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLExceptions(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	return dsn
}

func TestMySQLV7SnapshotRoundTripPreservesExceptionRows(t *testing.T) {
	ctx := context.Background()
	dsn := schema7ExceptionSnapshotFixture(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.snapshot_marker',CAST('"v7"' AS JSON)) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v7-snapshot.json")
	report, err := ExportMySQLSnapshot(ctx, dsn, path)
	if err != nil || report.SourceSchemaVersion != 7 {
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
	var envelope snapshotV7
	if json.Unmarshal(data, &envelope) != nil || envelope.Version != 7 || len(envelope.Exceptions) < 1 {
		t.Fatal("envelope", envelope.Version, len(envelope.Exceptions))
	}
	if !strings.Contains(string(envelope.StateRoot), "snapshot_marker") {
		t.Fatal("root extension lost")
	}
	target := testsupport.MySQLDSN(t)
	restored, err := ImportMySQLSnapshot(ctx, target, path, report.SHA256, 7)
	if err != nil || restored.SourceSchemaVersion != 7 {
		t.Fatal("import", restored, err)
	}
	again := filepath.Join(t.TempDir(), "re-export.json")
	report2, err := ExportMySQLSnapshot(ctx, target, again)
	if err != nil || report2.SourceSchemaVersion != 7 {
		t.Fatal("re-export", err)
	}
	var a, b snapshotV7
	first, _ := os.ReadFile(path)
	second, _ := os.ReadFile(again)
	json.Unmarshal(first, &a)
	json.Unmarshal(second, &b)
	if !equalApplicationJSON(a.StateRoot, b.StateRoot) || len(a.Exceptions) != len(b.Exceptions) {
		t.Fatal("snapshot drifted")
	}
	byID := map[string]json.RawMessage{}
	for _, row := range a.Exceptions {
		byID[row.ExceptionID] = row.Payload
	}
	for _, row := range b.Exceptions {
		if !equalApplicationJSON(byID[row.ExceptionID], row.Payload) {
			t.Fatal("exception raw drifted", row.ExceptionID)
		}
	}
	opened, err := OpenMySQL(target)
	if err != nil {
		t.Fatal("runtime rejected schema7 import", err)
	}
	opened.Close()
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',JSON_ARRAY()) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = ExportMySQLSnapshot(ctx, dsn, filepath.Join(t.TempDir(), "bad.json")); err == nil {
		t.Fatal("ambiguous root exported")
	}
}