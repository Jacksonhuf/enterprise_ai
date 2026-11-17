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

func schema6SettingsSnapshotFixture(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	dsn, _, _ := schema5SettingsRoot(t)
	plan, err := PlanMySQLSettings(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLSettings(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	return dsn
}

func TestMySQLV6SnapshotRoundTripPreservesSettingsRows(t *testing.T) {
	ctx := context.Background()
	dsn := schema6SettingsSnapshotFixture(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.snapshot_marker',CAST('"v6"' AS JSON)) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v6-snapshot.json")
	report, err := ExportMySQLSnapshot(ctx, dsn, path)
	if err != nil || report.SourceSchemaVersion != 6 {
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
	var envelope snapshotV6
	if json.Unmarshal(data, &envelope) != nil || envelope.Version != 6 || len(envelope.Settings) < 1 {
		t.Fatal("envelope", envelope.Version, len(envelope.Settings))
	}
	if !strings.Contains(string(envelope.StateRoot), "snapshot_marker") {
		t.Fatal("root extension lost")
	}
	target := testsupport.MySQLDSN(t)
	restored, err := ImportMySQLSnapshot(ctx, target, path, report.SHA256, 6)
	if err != nil || restored.SourceSchemaVersion != 6 {
		t.Fatal("import", restored, err)
	}
	again := filepath.Join(t.TempDir(), "re-export.json")
	report2, err := ExportMySQLSnapshot(ctx, target, again)
	if err != nil || report2.SourceSchemaVersion != 6 {
		t.Fatal("re-export", err)
	}
	var a, b snapshotV6
	first, _ := os.ReadFile(path)
	second, _ := os.ReadFile(again)
	json.Unmarshal(first, &a)
	json.Unmarshal(second, &b)
	if !equalApplicationJSON(a.StateRoot, b.StateRoot) || len(a.Settings) != len(b.Settings) {
		t.Fatal("snapshot drifted")
	}
	byKey := map[string]json.RawMessage{}
	for _, row := range a.Settings {
		byKey[row.SettingKey] = row.Payload
	}
	for _, row := range b.Settings {
		if !equalApplicationJSON(byKey[row.SettingKey], row.Payload) {
			t.Fatal("settings raw drifted", row.SettingKey)
		}
	}
	opened, err := OpenMySQL(target)
	if err != nil {
		t.Fatal("runtime rejected schema6 import", err)
	}
	opened.Close()
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.settings',JSON_OBJECT()) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = ExportMySQLSnapshot(ctx, dsn, filepath.Join(t.TempDir(), "bad.json")); err == nil {
		t.Fatal("ambiguous root exported")
	}
}