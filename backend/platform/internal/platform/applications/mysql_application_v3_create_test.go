package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"reflect"
	"testing"
)

func TestMySQLV3ApplicationCreationTemplateBindingAndAtomicity(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	s, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	db := preflightDB(t, dsn)
	if err = createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = stageMySQLApplicationRoot(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE cognara_fg_state SET schema_version=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	before, err := s.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	source := before.Apps["channel-replenishment"]
	raw, _ := json.Marshal(source.Draft)
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	fields["future"] = json.RawMessage(`{"n":9007199254740993,"decimal":0.1234567890123456789}`)
	raw, _ = json.Marshal(fields)
	if _, err = db.Exec(`UPDATE cognara_fg_application_drafts SET config=?,raw_payload=? WHERE app_id=?`, string(raw), raw, source.ID); err != nil {
		t.Fatal(err)
	}
	cfg := source.Draft
	cfg.Basic = Basic{Name: "复制应用", Description: "new"}
	stale := "stale"
	if _, err = createMySQLV3Application(ctx, db, source.TenantID, "owner", cfg, source.ID, &stale); !errors.Is(err, errRelationalDraftConflict) {
		t.Fatal("stale template accepted", err)
	}
	if _, err = createMySQLV3Application(ctx, db, "foreign", "owner", cfg, source.ID, &stale); !errors.Is(err, errRelationalAppUnavailable) {
		t.Fatal("tenant checked after revision", err)
	}
	copied, err := createMySQLV3Application(ctx, db, source.TenantID, "owner", cfg, source.ID, &source.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if copied.ID == source.ID || copied.AgentID == source.AgentID || copied.CurrentReleaseID != nil || len(copied.Releases) != 0 || copied.OwnerID != "owner" {
		t.Fatal("copy inherited identity/publication")
	}
	var stored []byte
	if err = db.QueryRow(`SELECT raw_payload FROM cognara_fg_application_drafts WHERE app_id=?`, copied.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	json.Unmarshal(stored, &got)
	if !equalApplicationJSON(fields["future"], got["future"]) {
		t.Fatal("template precision/extension lost")
	}
	blank, err := createMySQLV3Application(ctx, db, source.TenantID, "owner", cfg, "", nil)
	if err != nil || blank.Draft.Basic.Name != cfg.Basic.Name {
		t.Fatal("blank create failed", err)
	}
	// Legacy template clients may omit the preview revision.
	if _, err = createMySQLV3Application(ctx, db, source.TenantID, "owner", cfg, source.ID, nil); err != nil {
		t.Fatal(err)
	}
	after, err := s.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Apps) != len(before.Apps)+3 || len(after.Audit) != len(before.Audit)+3 || !reflect.DeepEqual(before.Grants, after.Grants) || !reflect.DeepEqual(before.Sessions, after.Sessions) || !reflect.DeepEqual(source, after.Apps[source.ID]) {
		t.Fatal("creation changed unrelated state")
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_create_audit BEFORE INSERT ON cognara_fg_audit FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected create failure'`); err != nil {
		t.Fatal(err)
	}
	if a, e := createMySQLV3Application(ctx, db, source.TenantID, "owner", cfg, "", nil); e == nil || a.ID != "" {
		t.Fatal("failed create returned app")
	}
	rolled, err := s.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, rolled) {
		t.Fatal("partial create persisted")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := preflightDB(t, dsn)
	tx, err = reopened.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	state, _, err := readMySQLApplicationV3(ctx, tx, true)
	if err != nil || state.Apps[copied.ID].Revision != copied.Revision {
		t.Fatal("reopen lost created app", err)
	}
}
