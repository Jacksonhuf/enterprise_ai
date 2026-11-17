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

func TestMySQLV5SnapshotRoundTripPreservesSessionRows(t *testing.T) {
	ctx := context.Background()
	dsn := schema5SessionFixture(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.snapshot_marker',CAST('"v5"' AS JSON)) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v5-snapshot.json")
	report, err := ExportMySQLSnapshot(ctx, dsn, path)
	if err != nil || report.SourceSchemaVersion != 5 {
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
	var envelope snapshotV5
	if json.Unmarshal(data, &envelope) != nil || envelope.Version != 5 || len(envelope.Sessions) < 1 {
		t.Fatal("envelope", envelope.Version, len(envelope.Sessions))
	}
	if !strings.Contains(string(envelope.StateRoot), "snapshot_marker") {
		t.Fatal("root extension lost")
	}
	target := testsupport.MySQLDSN(t)
	restored, err := ImportMySQLSnapshot(ctx, target, path, report.SHA256, 5)
	if err != nil || restored.SourceSchemaVersion != 5 {
		t.Fatal("import", restored, err)
	}
	again := filepath.Join(t.TempDir(), "re-export.json")
	report2, err := ExportMySQLSnapshot(ctx, target, again)
	if err != nil || report2.SourceSchemaVersion != 5 {
		t.Fatal("re-export", err)
	}
	var a, b snapshotV5
	first, _ := os.ReadFile(path)
	second, _ := os.ReadFile(again)
	json.Unmarshal(first, &a)
	json.Unmarshal(second, &b)
	if !equalApplicationJSON(a.StateRoot, b.StateRoot) || len(a.Sessions) != len(b.Sessions) {
		t.Fatal("snapshot drifted")
	}
	byID := map[string]json.RawMessage{}
	for _, row := range a.Sessions {
		byID[row.SessionID] = row.Payload
	}
	for _, row := range b.Sessions {
		if !equalApplicationJSON(byID[row.SessionID], row.Payload) {
			t.Fatal("session raw drifted", row.SessionID)
		}
	}
	opened, err := OpenMySQL(target)
	if err != nil {
		t.Fatal("runtime rejected schema5 import", err)
	}
	opened.Close()
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.sessions',JSON_OBJECT()) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = ExportMySQLSnapshot(ctx, dsn, filepath.Join(t.TempDir(), "bad.json")); err == nil {
		t.Fatal("ambiguous root exported")
	}
}