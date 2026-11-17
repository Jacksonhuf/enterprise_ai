package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLApplicationCheckpointRestoresViaExistingV2Importer(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	if err = store.UpdateOperational(func(st *State) error {
		st.Audit = append(st.Audit, Event{ID: "checkpoint-event", TenantID: "42", OccurredAt: "2026-11-17T00:00:00Z", Action: "checkpoint"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	db := preflightDB(t, dsn)
	original := rawLayoutFixture(t)
	commitApplicationLayout(t, db, original)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	checkpoint, count, err := exportMySQLApplicationCheckpoint(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("audit omitted", count)
	}
	state, audit, events, err := parseMySQLV2(checkpoint)
	if err != nil || !equalApplicationJSON(original, state) || len(audit) != 1 || len(events) != 1 || events[0].ID != "checkpoint-event" {
		t.Fatal("checkpoint lost raw state or audit", err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "application-checkpoint.json")
	if err = writeNewSnapshot(path, checkpoint); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(checkpoint)
	target := testsupport.MySQLDSN(t)
	report, err := ImportMySQLSnapshot(ctx, target, path, hex.EncodeToString(sum[:]), 2)
	if err != nil || report.AuditEvents != 1 {
		t.Fatal("existing importer rejected checkpoint", report, err)
	}
	restored := preflightDB(t, target)
	var got, want []byte
	if err = restored.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	// v2 restoration explicitly normalizes JSON. Compare against that documented
	// database representation, not against the binary envelopes' number lexemes.
	if err = db.QueryRow(`SELECT CAST(? AS JSON)`, string(original)).Scan(&want); err != nil {
		t.Fatal(err)
	}
	if !equalApplicationJSON(want, got) {
		t.Fatal("restored normalized state differs")
	}
	exported := filepath.Join(t.TempDir(), "restored.json")
	if _, err = ExportMySQLSnapshot(ctx, target, exported); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(exported)
	if err != nil {
		t.Fatal(err)
	}
	restoredState, _, restoredEvents, err := parseMySQLV2(again)
	if err != nil || !equalApplicationJSON(got, restoredState) || len(restoredEvents) != 1 || restoredEvents[0].ID != "checkpoint-event" {
		t.Fatal("restore/re-export differs", err)
	}
	// Corruption must produce no successful checkpoint bytes.
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE cognara_fg_application_drafts SET revision='wrong'`); err != nil {
		t.Fatal(err)
	}
	bad, _, err := exportMySQLApplicationCheckpoint(ctx, tx)
	if err == nil || len(bad) != 0 {
		t.Fatal("corrupt relational state exported")
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if bad, _, err = exportMySQLApplicationCheckpoint(canceled, tx); err == nil || len(bad) != 0 {
		t.Fatal("canceled checkpoint succeeded")
	}
}
