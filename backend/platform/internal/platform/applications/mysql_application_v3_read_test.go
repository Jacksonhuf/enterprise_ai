package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLApplicationV3ReaderPersistsAndKeepsRuntimeClosed(t *testing.T) {
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
		st.Audit = append(st.Audit, Event{ID: "v3-reader", TenantID: "42", OccurredAt: "2026-11-17T00:00:00Z", Action: "reader"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	db := preflightDB(t, dsn)
	if err = createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	var before []byte
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err = readMySQLApplicationV3(ctx, tx, true); err == nil {
		t.Fatal("v2 accepted")
	}
	if err = stageMySQLApplicationRoot(ctx, tx); err != nil {
		t.Fatal(err)
	}
	// Activate only this isolated fixture; no production migration entry exists.
	if _, err = tx.Exec(`UPDATE cognara_fg_state SET schema_version=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db = preflightDB(t, dsn)
	for _, audit := range []bool{false, true} {
		tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
		if err != nil {
			t.Fatal(err)
		}
		st, raw, e := readMySQLApplicationV3(ctx, tx, audit)
		if e != nil {
			tx.Rollback()
			t.Fatal(e)
		}
		if !equalApplicationJSON(before, raw) || len(st.Apps) != 1 || len(st.Audit) != map[bool]int{false: 0, true: 1}[audit] {
			tx.Rollback()
			t.Fatal("reader changed state/audit")
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if opened, e := OpenMySQL(dsn); e == nil {
		opened.Close()
		t.Fatal("runtime opened incomplete schema3")
	}
	invoked := false
	if err = store.UpdateOperational(func(st *State) error { invoked = true; return nil }); err == nil || invoked {
		t.Fatal("legacy writer accepted schema3", err)
	}
	// One snapshot sees the same root even when another connection updates it.
	tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, _, err = readMySQLApplicationV3(ctx, tx, false); err != nil {
		t.Fatal(err)
	}
	other := preflightDB(t, dsn)
	if _, err = other.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future','new-root') WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	_, raw, err := readMySQLApplicationV3(ctx, tx, false)
	if err != nil || !equalApplicationJSON(before, raw) {
		t.Fatal("inconsistent read snapshot", err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	_, raw, err = readMySQLApplicationV3(ctx, tx, false)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	if string(fields["future"]) != `"new-root"` {
		t.Fatal("new transaction used old root")
	}
	if _, err = tx.Exec(`UPDATE cognara_fg_application_envelope SET root_payload='{"duplicate":true}' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, raw, err = readMySQLApplicationV3(ctx, tx, false); err == nil || raw != nil {
		t.Fatal("corruption returned successful data")
	}
}
