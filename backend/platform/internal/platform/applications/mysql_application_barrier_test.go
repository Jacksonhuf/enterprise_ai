package applications

import (
	"context"
	"database/sql"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
	"time"
)

func TestMySQLApplicationMigrationBarrierUsesLockedSourceAndRollsBack(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	db := preflightDB(t, dsn)
	if err = createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = rehearseMySQLApplicationMigration(ctx, db); err == nil {
		t.Fatal("schema1 accepted")
	}
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	var before, after []byte
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = rehearseMySQLApplicationMigration(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); err != nil || string(before) != string(after) {
		t.Fatal("source changed", err)
	}
	for table := range applicationSchemaSpecs() {
		var count int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("rehearsal persisted target", table, count, err)
		}
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = copyMySQLApplicationSource(ctx, tx); err != nil {
		t.Fatal(err)
	}
	// A second pool cannot write the source until the migration transaction ends.
	other := preflightDB(t, dsn)
	timeout, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	_, err = other.ExecContext(timeout, `UPDATE cognara_fg_state SET body=body WHERE id=1`)
	cancel()
	if err == nil {
		t.Fatal("source writer bypassed migration barrier")
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err = other.ExecContext(ctx, `UPDATE cognara_fg_state SET body=JSON_SET(body,'$.migration_test','latest') WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = copyMySQLApplicationSource(ctx, tx); err != nil {
		t.Fatal(err)
	}
	got, err := loadMySQLApplicationLayout(ctx, tx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); err != nil || !equalApplicationJSON(after, got) {
		t.Fatal("did not copy current locked source", err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_barrier_copy BEFORE INSERT ON cognara_fg_application_drafts FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected barrier failure'`); err != nil {
		t.Fatal(err)
	}
	if err = rehearseMySQLApplicationMigration(ctx, db); err == nil {
		t.Fatal("copy failure accepted")
	}
	for table := range applicationSchemaSpecs() {
		var n int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial barrier copy", table, n, err)
		}
	}
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); err != nil || string(before) != string(after) {
		t.Fatal("failure modified source", err)
	}
	if _, err = db.Exec(`DROP TRIGGER fail_barrier_copy`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE cognara_fg_state SET schema_version=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = rehearseMySQLApplicationMigration(ctx, db); err == nil {
		t.Fatal("unsupported source accepted")
	}
}
