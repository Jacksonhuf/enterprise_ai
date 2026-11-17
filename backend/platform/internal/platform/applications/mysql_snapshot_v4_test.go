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

func TestMySQLV4SnapshotRoundTripPreservesGrantRows(t *testing.T) {
	ctx := context.Background()
	dsn := schema4GrantFixture(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.snapshot_marker',CAST('"v4"' AS JSON)) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v4-snapshot.json")
	report, err := ExportMySQLSnapshot(ctx, dsn, path)
	if err != nil || report.SourceSchemaVersion != 4 {
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
	var envelope snapshotV4
	if json.Unmarshal(data, &envelope) != nil || envelope.Version != 4 || len(envelope.Grants) < 1 {
		t.Fatal("envelope", envelope.Version, len(envelope.Grants))
	}
	if !strings.Contains(string(envelope.StateRoot), "snapshot_marker") {
		t.Fatal("root extension lost")
	}
	target := testsupport.MySQLDSN(t)
	restored, err := ImportMySQLSnapshot(ctx, target, path, report.SHA256, 4)
	if err != nil || restored.SourceSchemaVersion != 4 {
		t.Fatal("import", restored, err)
	}
	again := filepath.Join(t.TempDir(), "re-export.json")
	report2, err := ExportMySQLSnapshot(ctx, target, again)
	if err != nil || report2.SourceSchemaVersion != 4 {
		t.Fatal("re-export", err)
	}
	var a, b snapshotV4
	first, _ := os.ReadFile(path)
	second, _ := os.ReadFile(again)
	json.Unmarshal(first, &a)
	json.Unmarshal(second, &b)
	if !equalApplicationJSON(a.StateRoot, b.StateRoot) || len(a.Grants) != len(b.Grants) {
		t.Fatal("snapshot drifted")
	}
	byID := map[string]json.RawMessage{}
	for _, row := range a.Grants {
		byID[row.GrantID] = row.Payload
	}
	for _, row := range b.Grants {
		if !equalApplicationJSON(byID[row.GrantID], row.Payload) {
			t.Fatal("grant raw drifted", row.GrantID)
		}
	}
	opened, err := OpenMySQL(target)
	if err != nil {
		t.Fatal("runtime rejected schema4 import", err)
	}
	opened.Close()
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.grants',JSON_ARRAY()) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = ExportMySQLSnapshot(ctx, dsn, filepath.Join(t.TempDir(), "bad.json")); err == nil {
		t.Fatal("ambiguous root exported")
	}
}