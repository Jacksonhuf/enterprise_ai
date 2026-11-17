package applications

import (
	"context"
	"database/sql"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMySQLApplicationV3LegacyReadProjectionsAndWriteRejection(t *testing.T) {
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
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.settings.test',JSON_OBJECT('theme','dark'),'$.setting_revisions.test','revision','$.exceptions',JSON_ARRAY(JSON_OBJECT('id','exception','tenant_id','42')),'$.usage',JSON_ARRAY(JSON_OBJECT('id','usage','tenant_id','42'))) WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = s.UpdateOperational(func(st *State) error {
		st.Audit = append(st.Audit, Event{ID: "query-event", TenantID: "42", OccurredAt: "2026-11-17T00:00:00Z", Action: "query"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Typed updates may normalize fixtures. Capture the actual schema2 baseline.
	settings, handled, err := s.ReadMySQLSettingsContext(ctx, "test", "")
	if err != nil || !handled {
		t.Fatal(err)
	}
	match := func(map[string]any) bool { return true }
	id := func(v map[string]any) string { return v["id"].(string) }
	exceptions, _, _, err := s.QueryMySQLExceptions(ctx, "42", "", 100, match, id)
	if err != nil {
		t.Fatal(err)
	}
	usage, _, _, err := s.QueryMySQLUsage(ctx, "42", "", 100, match, id)
	if err != nil {
		t.Fatal(err)
	}
	events, _, _, err := s.QueryMySQLAudit(ctx, "42", "", 100, func(Event) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
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
	got, h, e := s.ReadMySQLSettingsContext(ctx, "test", "")
	if e != nil || !h || !reflect.DeepEqual(got, settings) {
		t.Fatal("settings read differs", e)
	}
	a, _, h, e := s.QueryMySQLExceptions(ctx, "42", "", 100, match, id)
	if e != nil || !h || !reflect.DeepEqual(a, exceptions) {
		t.Fatal("exception read differs", e)
	}
	b, _, h, e := s.QueryMySQLUsage(ctx, "42", "", 100, match, id)
	if e != nil || !h || !reflect.DeepEqual(b, usage) {
		t.Fatal("usage read differs", e)
	}
	c, _, h, e := s.QueryMySQLAudit(ctx, "42", "", 100, func(Event) bool { return true })
	if e != nil || !h || !reflect.DeepEqual(c, events) {
		t.Fatal("audit read differs", e)
	}
	foreign, _, h, e := s.QueryMySQLExceptions(ctx, "43", "", 100, match, id)
	if e != nil || !h || len(foreign) != 0 {
		t.Fatal("cross-tenant exception leakage", e)
	}
	for _, alias := range []string{"Settings", "Usage"} {
		if _, err = db.Exec("UPDATE cognara_fg_state SET body=JSON_SET(body,?,JSON_OBJECT()) WHERE id=1", "$."+alias); err != nil {
			t.Fatal(err)
		}
		if alias == "Settings" {
			if _, _, e = s.ReadMySQLSettingsContext(ctx, "test", ""); e == nil {
				t.Fatal("settings alias fallback")
			}
		} else {
			if _, _, _, e = s.QueryMySQLUsage(ctx, "42", "", 100, match, id); e == nil {
				t.Fatal("usage alias fallback")
			}
		}
		if _, err = db.Exec("UPDATE cognara_fg_state SET body=JSON_REMOVE(body,?) WHERE id=1", "$."+alias); err != nil {
			t.Fatal(err)
		}
	}
	full, e := s.readSQLContext(ctx)
	if e != nil || len(full.Apps) != 1 || len(full.Audit) != len(events) {
		t.Fatal("general read incomplete", e)
	}
	operational, e := s.readSQLContextAudit(ctx, false)
	if e != nil || len(operational.Apps) != 1 || len(operational.Audit) != 0 {
		t.Fatal("operational read incomplete", e)
	}
	if e = s.Check(); e != nil {
		t.Fatal("valid relational readiness rejected", e)
	}
	snapshot := filepath.Join(t.TempDir(), "unsupported-v3.json")
	if _, e = ExportMySQLSnapshot(ctx, dsn, snapshot); e == nil {
		t.Fatal("legacy snapshot accepted schema3")
	}
	if _, e = os.Stat(snapshot); !os.IsNotExist(e) {
		t.Fatal("unsupported snapshot published a file", e)
	}
	if h, e = s.AppendMySQLAuditContext(ctx, []Event{{ID: "rejected-write", TenantID: "42", OccurredAt: "2026-11-17T00:00:00Z", Action: "blocked"}}); e == nil || !h {
		t.Fatal("legacy audit writer enabled")
	}
	exceptionInvoked := false
	if _, h, e = s.UpdateMySQLExceptionContext(ctx, "42", "exception", func(map[string]any) (*ExceptionPatch, error) { exceptionInvoked = true; return nil, nil }); e == nil || !h || exceptionInvoked {
		t.Fatal("legacy exception writer enabled")
	}
	invoked := false
	h, e = s.UpdateMySQLSettingsContext(ctx, "test", "", func(SettingsRecord) (*SettingsPatch, error) { invoked = true; return nil, nil })
	if e == nil || !h || invoked {
		t.Fatal("legacy settings writer enabled")
	}
	if opened, e := OpenMySQL(dsn); e == nil {
		opened.Close()
		t.Fatal("runtime enabled")
	}
	if _, err = db.Exec(`UPDATE cognara_fg_application_envelope SET root_payload='{"duplicate":true}' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, e = s.readSQLContext(ctx); e == nil {
		t.Fatal("general read ignored corruption")
	}
	if e = s.Check(); e == nil {
		t.Fatal("readiness ignored corruption")
	}
	if _, _, e = s.ReadMySQLSettingsContext(ctx, "test", ""); e == nil {
		t.Fatal("settings bypassed schema3 integrity")
	}
	if _, _, _, e = s.QueryMySQLExceptions(ctx, "42", "", 100, match, id); e == nil {
		t.Fatal("exceptions bypassed integrity")
	}
	if _, _, _, e = s.QueryMySQLUsage(ctx, "42", "", 100, match, id); e == nil {
		t.Fatal("usage bypassed integrity")
	}
	if _, _, _, e = s.QueryMySQLAudit(ctx, "42", "", 100, func(Event) bool { return true }); e == nil {
		t.Fatal("audit bypassed integrity")
	}
}
