package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func importFile(t *testing.T, raw []byte) (string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	if e := os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw)
	return path, hex.EncodeToString(sum[:])
}
func TestSnapshotImportRejectsInvalidSourceBeforeConnecting(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `{"apps":{},"sessions":{},"apps":{}}`, `{"apps":{},"sessions":{},"extra":{"x":1,"x":2}}`, `{"apps":{},"sessions":{}} {}`, `{"apps":{},"sessions":{"s":{"session_id":"s","app_id":"missing","tenant_id":"42","owner_id":"user"}}}`} {
		path, sha := importFile(t, []byte(raw))
		if _, e := ImportMySQLSnapshot(context.Background(), "invalid", path, sha, 1); e == nil || !strings.Contains(e.Error(), "snapshot") {
			t.Fatal("bad source accepted")
		}
		if e := validateSnapshotJSON([]byte(raw)); e == nil && (raw == `{"apps":{},"sessions":{},"apps":{}}` || raw == `{"apps":{},"sessions":{},"extra":{"x":1,"x":2}}`) {
			t.Fatal("duplicate keys accepted")
		}
	}
	path, sha := importFile(t, []byte(`{"apps":{},"sessions":{}}`))
	if _, e := ImportMySQLSnapshot(context.Background(), "", path, sha, 2); e == nil {
		t.Fatal("unknown version")
	}
	if _, e := ImportMySQLSnapshot(context.Background(), "", path, "00", 1); e == nil {
		t.Fatal("invalid digest")
	}
}
func snapshotSource(t *testing.T) (string, string) {
	t.Helper()
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Update(func(st *State) error {
		st.Sessions["transfer"] = Session{ID: "transfer", AppID: "channel-replenishment", TenantID: "42", OwnerID: "demo-user", Title: "迁移验证", Messages: []Message{{ID: "m-transfer", SessionID: "transfer", Content: "keep", Completed: true}}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "source.json")
	r, e := ExportMySQLSnapshot(context.Background(), dsn, path)
	if e != nil {
		t.Fatal(e)
	}
	return path, r.SHA256
}
func TestMySQLSnapshotImportConcurrentAndReopen(t *testing.T) {
	path, sha := snapshotSource(t)
	target := testsupport.MySQLDSN(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := ImportMySQLSnapshot(context.Background(), target, path, sha, 1)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for e := range results {
		if e == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatal("import must have one winner", successes)
	}
	s, e := OpenMySQL(target)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if s.Read().Sessions["transfer"].Title != "迁移验证" {
		t.Fatal("source data lost")
	}
	if _, e = ImportMySQLSnapshot(context.Background(), target, path, sha, 1); e == nil {
		t.Fatal("existing target overwritten")
	}
	output := filepath.Join(t.TempDir(), "target.json")
	if _, e = ExportMySQLSnapshot(context.Background(), target, output); e != nil {
		t.Fatal(e)
	}
	left, _ := os.ReadFile(path)
	right, _ := os.ReadFile(output)
	var a, b any
	json.Unmarshal(left, &a)
	json.Unmarshal(right, &b)
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	if string(aj) != string(bj) {
		t.Fatal("round trip changed state")
	}
}
func TestCompiledMySQLSnapshotImport(t *testing.T) {
	binary := os.Getenv("COGNARA_TEST_IMPORT_BINARY")
	if binary == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("compiled importer required")
		}
		t.Skip("compiled importer not configured")
	}
	path, sha := snapshotSource(t)
	target := testsupport.MySQLDSN(t)
	run := func() ([]byte, error) {
		cmd := exec.Command(binary, "--mysql-dsn-env", "COGNARA_IMPORT_TARGET_DSN", "--input", path, "--expected-sha256", sha, "--source-schema-version", "1")
		cmd.Env = append(os.Environ(), "COGNARA_IMPORT_TARGET_DSN="+target)
		return cmd.CombinedOutput()
	}
	raw, e := run()
	if e != nil {
		t.Fatal("compiled import failed")
	}
	var report SnapshotReport
	if json.Unmarshal(raw, &report) != nil || report.SHA256 != sha || report.Sessions != 1 {
		t.Fatal("invalid compiled import report")
	}
	if _, e = run(); e == nil {
		t.Fatal("compiled importer overwrote target")
	}
	s, e := OpenMySQL(target)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if s.Read().Sessions["transfer"].Messages[0].Content != "keep" {
		t.Fatal("compiled import lost state")
	}
}

func TestMySQLRejectedImportLeavesEmptyDatabase(t *testing.T) {
	target := testsupport.MySQLDSN(t)
	path, _ := snapshotSource(t)
	if _, e := ImportMySQLSnapshot(context.Background(), target, path, strings.Repeat("0", 64), 1); e == nil {
		t.Fatal("checksum mismatch accepted")
	}
	db, e := sql.Open("mysql", target)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var count int
	if e = db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()").Scan(&count); e != nil || count != 0 {
		t.Fatal("invalid source changed target", e, count)
	}
}
