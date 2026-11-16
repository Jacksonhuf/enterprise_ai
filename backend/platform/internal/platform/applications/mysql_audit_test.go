package applications

import (
	"context"
	"database/sql"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
)

func TestMySQLRelationalAuditMigrationAndAtomicUpdates(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	first := Event{ID: "event-one", TenantID: "42", Action: "test", OccurredAt: Now()}
	if e = s.Update(func(st *State) error { st.Audit = append(st.Audit, first); return nil }); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal("idempotent migration", e)
	}
	reopened, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if len(reopened.Read().Audit) != 1 || reopened.Read().Audit[0].ID != first.ID {
		t.Fatal("audit lost")
	}
	if e = reopened.Update(func(st *State) error {
		st.Settings["atomic"] = []byte(`"committed"`)
		st.Audit = append(st.Audit, Event{ID: "event-two", TenantID: "42", Action: "save", OccurredAt: Now()})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	// A duplicate relational event rolls back both state and the audit insert.
	if e = reopened.Update(func(st *State) error {
		st.Settings["atomic"] = []byte(`"discard"`)
		st.Audit = append(st.Audit, first)
		return nil
	}); e == nil {
		t.Fatal("duplicate committed")
	}
	if string(s.Read().Settings["atomic"]) != `"committed"` || len(s.Read().Audit) != 2 {
		t.Fatal("partial transaction")
	}
	if e = s.Update(func(st *State) error { st.Audit[0].Action = "tamper"; return nil }); e == nil {
		t.Fatal("audit modified")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var version, count int
	if e = db.QueryRow("SELECT schema_version FROM cognara_fg_state WHERE id=1").Scan(&version); e != nil || version != 2 {
		t.Fatal("wrong version", e)
	}
	if e = db.QueryRow("SELECT COUNT(*) FROM cognara_fg_audit WHERE tenant_id='42'").Scan(&count); e != nil || count != 2 {
		t.Fatal("wrong relational rows", e, count)
	}
}
func TestMySQLRelationalAuditMigrationFailurePreservesSource(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	event := Event{ID: "duplicate", TenantID: "42", OccurredAt: Now()}
	if e = s.Update(func(st *State) error { st.Audit = []Event{event, event}; return nil }); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e == nil {
		t.Fatal("duplicate source migrated")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var version, count int
	if e = db.QueryRow("SELECT schema_version FROM cognara_fg_state WHERE id=1").Scan(&version); e != nil || version != 1 {
		t.Fatal("version changed")
	}
	if e = db.QueryRow("SELECT COUNT(*) FROM cognara_fg_audit").Scan(&count); e != nil || count != 0 {
		t.Fatal("partial migration")
	}
	if len(s.Read().Audit) != 2 {
		t.Fatal("source lost")
	}
}
