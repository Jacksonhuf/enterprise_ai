package applications

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

// TestR20Schema8RunbookSnapshotImportDrill mirrors the production runbook rollback path:
// export schema8 after PrepareSchema8IntegrationDB, import into an isolated DSN, dual OpenMySQL read.
func TestR20Schema8RunbookSnapshotImportDrill(t *testing.T) {
	ctx := context.Background()
	source := testsupport.MySQLDSN(t)
	if err := PrepareSchema8IntegrationDB(ctx, source); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "runbook-v8-snapshot.json")
	report, err := ExportMySQLSnapshot(ctx, source, path)
	if err != nil || report.SourceSchemaVersion != 8 {
		t.Fatal("export", report, err)
	}
	target := testsupport.MySQLDSN(t)
	restored, err := ImportMySQLSnapshot(ctx, target, path, report.SHA256, 8)
	if err != nil || restored.SourceSchemaVersion != 8 {
		t.Fatal("import", restored, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("snapshot file", err)
	}
	a, err := OpenMySQL(target)
	if err != nil {
		t.Fatal("open a", err)
	}
	b, err := OpenMySQL(target)
	if err != nil {
		a.Close()
		t.Fatal("open b", err)
	}
	defer a.Close()
	defer b.Close()
	left := len(a.Read().Usage)
	right := len(b.Read().Usage)
	if left < 1 || left != right {
		t.Fatal("usage rows", left, right)
	}
	var version int
	if err = preflightDB(t, target).QueryRow(`SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil || version != 8 {
		t.Fatal("schema_version", version, err)
	}
}