package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"os"
	"path/filepath"
	"testing"
)

func TestMySQLSnapshotPreservesDataAndRefusesOverwrite(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Update(func(st *State) error {
		st.Sessions["export-session"] = Session{ID: "export-session", Title: "中文快照", Messages: []Message{{ID: "m1", Content: "验证内容"}}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	// Unknown top-level properties must survive schema evolution without typed round-tripping.
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_field','preserve') WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "snapshot.json")
	report, e := ExportMySQLSnapshot(context.Background(), dsn, path)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw)
	if report.SHA256 != hex.EncodeToString(sum[:]) || report.Bytes != len(raw) || report.Messages != 1 {
		t.Fatal("wrong snapshot summary", report)
	}
	var decoded map[string]json.RawMessage
	if e = json.Unmarshal(raw, &decoded); e != nil || string(decoded["future_field"]) != `"preserve"` {
		t.Fatal("unknown data lost", e)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("snapshot permissions", info.Mode())
	}
	if _, e = ExportMySQLSnapshot(context.Background(), dsn, path); e == nil {
		t.Fatal("overwrote existing snapshot")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(raw) {
		t.Fatal("snapshot changed")
	}
	link := filepath.Join(t.TempDir(), "link")
	if e = os.Symlink(path, link); e != nil {
		t.Fatal(e)
	}
	if _, e = ExportMySQLSnapshot(context.Background(), dsn, link); e == nil {
		t.Fatal("followed destination symlink")
	}
	var count int
	if e = db.QueryRow("SELECT COUNT(*) FROM cognara_fg_state").Scan(&count); e != nil || count != 1 {
		t.Fatal("export mutated source")
	}
}
func TestMySQLSnapshotDoesNotInitializeEmptyDatabase(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	path := filepath.Join(t.TempDir(), "empty.json")
	if _, e := ExportMySQLSnapshot(context.Background(), dsn, path); e == nil {
		t.Fatal("empty source accepted")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("created output for invalid source")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var count int
	if e = db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()").Scan(&count); e != nil || count != 0 {
		t.Fatal("export initialized source", e, count)
	}
}
func TestSnapshotFilePublicationRefusesExistingDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if e := writeNewSnapshot(path, []byte(`{"ok":true}`)); e != nil {
		t.Fatal(e)
	}
	if e := writeNewSnapshot(path, []byte("bad")); e == nil {
		t.Fatal("overwrite allowed")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != `{"ok":true}` {
		t.Fatal("file overwritten")
	}
}
