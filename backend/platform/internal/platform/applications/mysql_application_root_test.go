package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
)

func TestMySQLApplicationRootHandoffKeepsOneAuthorityAndRollsBack(t *testing.T) {
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
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_root','keep') WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	var before, after []byte
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err = rehearseMySQLApplicationRoot(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); err != nil || string(before) != string(after) {
		t.Fatal("rehearsal changed source", err)
	}
	for table := range applicationSchemaSpecs() {
		var n int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("target persisted", table, n, err)
		}
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = stageMySQLApplicationRoot(ctx, tx); err != nil {
		t.Fatal(err)
	}
	var root []byte
	if err = tx.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(root, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"apps", "checks"} {
		if _, exists := fields[key]; exists {
			t.Fatal("aggregate application authority remains", key)
		}
	}
	if string(fields["future_root"]) != `"keep"` {
		t.Fatal("root extension lost")
	}
	if err = tx.QueryRow(`SELECT root_payload FROM cognara_fg_application_envelope WHERE id=1`).Scan(&root); err != nil || !equalApplicationJSON(root, []byte(`{}`)) {
		t.Fatal("envelope still owns root", err)
	}
	if _, err = tx.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_root','latest') WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	got, err := loadStagedMySQLApplicationState(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(got, &fields)
	if string(fields["future_root"]) != `"latest"` {
		t.Fatal("reader used stale envelope root")
	}
	if _, err = tx.Exec(`UPDATE cognara_fg_application_envelope SET root_payload='{"future_root":"duplicate"}' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = loadStagedMySQLApplicationState(ctx, tx); err == nil {
		t.Fatal("duplicate root authority accepted")
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_root_handoff BEFORE UPDATE ON cognara_fg_application_envelope FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected root handoff failure'`); err != nil {
		t.Fatal(err)
	}
	if err = rehearseMySQLApplicationRoot(ctx, db); err == nil {
		t.Fatal("handoff failure accepted")
	}
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); err != nil || string(before) != string(after) {
		t.Fatal("partial root handoff", err)
	}
	for table := range applicationSchemaSpecs() {
		var n int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial target", table, n, err)
		}
	}
}
