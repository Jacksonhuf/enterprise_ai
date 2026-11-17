package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"sync"
	"testing"
	"time"
)

func TestRelationalDraftMergePreservesOpaqueValues(t *testing.T) {
	raw := rawLayoutFixture(t)
	layout, err := splitApplicationLayout(raw, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	source := layout.Apps[0].Draft
	var cfg Config
	json.Unmarshal(source, &cfg)
	cfg.Basic.Name = "updated"
	got, err := mergeRelationalDraft(source, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]json.RawMessage
	json.Unmarshal(source, &before)
	json.Unmarshal(got, &after)
	if !equalApplicationJSON(before["future_config"], after["future_config"]) {
		t.Fatal("opaque precision lost")
	}
	before["basic"], _ = json.Marshal(cfg.Basic)
	expected, _ := json.Marshal(before)
	if !equalApplicationJSON(expected, got) {
		t.Fatal("unexpected config change")
	}
}

func TestMySQLV3DraftWriterCASAuditRollbackAndReopen(t *testing.T) {
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
	a := before.Apps["channel-replenishment"]
	next := a.Draft
	next.Basic.Name = "new draft"
	if _, err = saveMySQLV3Draft(ctx, db, "foreign", a.ID, "actor", "", "", a.Revision, next); !errors.Is(err, errRelationalAppUnavailable) {
		t.Fatal("tenant bypass", err)
	}
	second := preflightDB(t, dsn)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, connection := range []*sql.DB{db, second} {
		wg.Add(1)
		go func(c *sql.DB) {
			defer wg.Done()
			_, e := saveMySQLV3Draft(ctx, c, a.TenantID, a.ID, "actor", "", "", a.Revision, next)
			results <- e
		}(connection)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, errRelationalDraftConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("CAS incorrect", success, conflict)
	}
	after, err := s.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	saved := after.Apps[a.ID]
	if saved.Revision == a.Revision || saved.Draft.Basic.Name != "new draft" || len(after.Audit) != len(before.Audit)+1 {
		t.Fatal("draft/audit not committed together")
	}
	releaseBefore, _ := json.Marshal(a.Releases)
	releaseAfter, _ := json.Marshal(saved.Releases)
	if !equalApplicationJSON(releaseBefore, releaseAfter) || *a.CurrentReleaseID != *saved.CurrentReleaseID {
		t.Fatal("save changed published state")
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_draft_audit BEFORE INSERT ON cognara_fg_audit FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected audit failure'`); err != nil {
		t.Fatal(err)
	}
	next.Basic.Name = "must rollback"
	if _, err = saveMySQLV3Draft(ctx, db, a.TenantID, a.ID, "actor", "", "", saved.Revision, next); err == nil {
		t.Fatal("audit failure accepted")
	}
	rolled, err := s.readSQLContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := json.Marshal(after)
	y, _ := json.Marshal(rolled)
	if !equalApplicationJSON(x, y) {
		t.Fatal("partial draft transaction")
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
	got, _, err := readMySQLApplicationV3(ctx, tx, true)
	if err != nil || got.Apps[a.ID].Revision != saved.Revision {
		t.Fatal("reopen lost saved draft", err)
	}
}

func TestRelationalDraftMergeKeepsMissingUnchangedFields(t *testing.T) {
	source := []byte(`{"future":{"n":9007199254740993}}`)
	got, err := mergeRelationalDraft(source, Config{})
	if err != nil || !equalApplicationJSON(source, got) {
		t.Fatal("unchanged missing fields materialized", err)
	}
	if _, err = mergeRelationalDraft([]byte(`{"Basic":{},"basic":{}}`), Config{}); err == nil {
		t.Fatal("ambiguous known field accepted")
	}
}

func TestRelationalDraftMergeNullableObject(t *testing.T) {
	source := []byte(`{"basic":null,"future":{"n":9007199254740993}}`)
	got, err := mergeRelationalDraft(source, Config{})
	if err != nil || !equalApplicationJSON(source, got) {
		t.Fatal("unchanged null object rewritten", err)
	}
	next := Config{Basic: Basic{Name: "changed"}}
	got, err = mergeRelationalDraft(source, next)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(got, &fields)
	var basic Basic
	if err = json.Unmarshal(fields["basic"], &basic); err != nil || basic.Name != "changed" {
		t.Fatal("null cannot become object", err)
	}
	if !equalApplicationJSON(fields["future"], []byte(`{"n":9007199254740993}`)) {
		t.Fatal("extension lost")
	}
}
