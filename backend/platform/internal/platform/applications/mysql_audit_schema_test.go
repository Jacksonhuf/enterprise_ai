package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLAuditRejectsPrefixInvisibleAndNarrowTextSchemas(t *testing.T) {
	snapshot, report := sourceV2Snapshot(t)
	variants := map[string]string{
		"primary-prefix":     `ALTER TABLE cognara_fg_audit DROP PRIMARY KEY, ADD PRIMARY KEY(tenant_id(8),event_id(64))`,
		"tenant-prefix":      `ALTER TABLE cognara_fg_audit DROP INDEX audit_tenant_sequence, ADD INDEX audit_tenant_sequence(tenant_id(8),sequence_no)`,
		"invisible-sequence": `ALTER TABLE cognara_fg_audit ALTER INDEX audit_sequence INVISIBLE`,
		"invisible-tenant":   `ALTER TABLE cognara_fg_audit ALTER INDEX audit_tenant_sequence INVISIBLE`,
		"latin1-actor":       `ALTER TABLE cognara_fg_audit MODIFY actor_id VARCHAR(255) CHARACTER SET latin1 NOT NULL`,
		"utf8mb3-action":     `ALTER TABLE cognara_fg_audit MODIFY action VARCHAR(255) CHARACTER SET utf8mb3 NOT NULL`,
		"latin1-resource":    `ALTER TABLE cognara_fg_audit MODIFY resource_id VARCHAR(255) CHARACTER SET latin1 NOT NULL`,
	}
	for name, alter := range variants {
		t.Run(name, func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			db := preflightDB(t, dsn)
			s, e := OpenMySQL(dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			if _, e = db.Exec(mysqlAuditDDL); e != nil {
				t.Fatal(e)
			}
			if _, e = db.Exec(alter); e != nil {
				t.Fatal("schema fixture", e)
			}
			var before, after string
			var version int
			if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); e != nil {
				t.Fatal(e)
			}
			var ddlBefore, ddlAfter string
			var table string
			if e = db.QueryRow(`SHOW CREATE TABLE cognara_fg_audit`).Scan(&table, &ddlBefore); e != nil {
				t.Fatal(e)
			}
			if _, e = CheckMySQLAuditMigration(context.Background(), dsn); e == nil {
				t.Fatal("preflight accepted incompatible audit schema")
			}
			if e = MigrateMySQLAudit(context.Background(), dsn); e == nil {
				t.Fatal("migration accepted incompatible audit schema")
			}
			if e = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &after); e != nil || version != 1 || before != after {
				t.Fatal("migration changed source", e)
			}
			// Runtime v2 must also refuse the table, rather than using altered semantics.
			if _, e = db.Exec(`UPDATE cognara_fg_state SET schema_version=2 WHERE id=1`); e != nil {
				t.Fatal(e)
			}
			if opened, e := OpenMySQL(dsn); e == nil {
				opened.Close()
				t.Fatal("runtime accepted incompatible audit schema")
			}
			if e = s.Check(); e == nil {
				t.Fatal("readiness accepted incompatible audit schema")
			}
			if _, e = ExportMySQLSnapshot(context.Background(), dsn, t.TempDir()+"/rejected.json"); e == nil {
				t.Fatal("export accepted incompatible audit schema")
			}
			// On an empty import target, schema rejection must not commit state or audit.
			if _, e = db.Exec(`DELETE FROM cognara_fg_state`); e != nil {
				t.Fatal(e)
			}
			if _, e = ImportMySQLSnapshot(context.Background(), dsn, snapshot, report.SHA256, 2); e == nil {
				t.Fatal("restore accepted incompatible audit schema")
			}
			var count int
			if e = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_state`).Scan(&count); e != nil || count != 0 {
				t.Fatal("partial restore state", e)
			}
			if e = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_audit`).Scan(&count); e != nil || count != 0 {
				t.Fatal("partial restore audit", e)
			}
			if e = db.QueryRow(`SHOW CREATE TABLE cognara_fg_audit`).Scan(&table, &ddlAfter); e != nil || ddlBefore != ddlAfter {
				t.Fatal("guard altered existing schema", e)
			}
		})
	}
}

func TestMySQLAuditUsesUTF8MB4DespiteDatabaseDefault(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	db := preflightDB(t, dsn)
	// The DSN database name comes from the isolated test helper, never user input.
	var database string
	if e := db.QueryRow(`SELECT DATABASE()`).Scan(&database); e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(database, "cognara_test_") || strings.Contains(database, "`") {
		t.Fatal("unsafe test database")
	}
	if _, e := db.Exec("ALTER DATABASE `" + database + "` CHARACTER SET latin1"); e != nil {
		t.Fatal(e)
	}
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	event := Event{TenantID: "42", ID: strings.Repeat("😀", 128), ActorID: "本体工程师😀", Action: "保存🚀", ResourceID: "物料😀", OccurredAt: "2026-11-16T00:00:00Z"}
	if e = s.Update(func(st *State) error { st.Audit = []Event{event}; return nil }); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal("migration under latin1 database default", e)
	}
	var actor, action, resource string
	if e = db.QueryRow(`SELECT actor_id,action,resource_id FROM cognara_fg_audit`).Scan(&actor, &action, &resource); e != nil || actor != event.ActorID || action != event.Action || resource != event.ResourceID {
		t.Fatal("Unicode projection changed", e)
	}
	if got := s.Read().Audit; !reflect.DeepEqual(got, []Event{event}) {
		t.Fatal("payload changed")
	}
	path := t.TempDir() + "/utf8.json"
	report, e := ExportMySQLSnapshot(context.Background(), dsn, path)
	if e != nil {
		t.Fatal(e)
	}
	target := testsupport.MySQLDSN(t)
	other := preflightDB(t, target)
	if e = other.QueryRow(`SELECT DATABASE()`).Scan(&database); e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(database, "cognara_test_") || strings.Contains(database, "`") {
		t.Fatal("unsafe target database")
	}
	if _, e = other.Exec("ALTER DATABASE `" + database + "` CHARACTER SET latin1"); e != nil {
		t.Fatal(e)
	}
	if _, e = ImportMySQLSnapshot(context.Background(), target, path, report.SHA256, 2); e != nil {
		t.Fatal("Unicode restore under latin1 default", e)
	}
	var payload []byte
	if e = other.QueryRow(`SELECT actor_id,action,resource_id,payload FROM cognara_fg_audit`).Scan(&actor, &action, &resource, &payload); e != nil {
		t.Fatal(e)
	}
	var decoded Event
	if json.Unmarshal(payload, &decoded) != nil || decoded != event || actor != event.ActorID || action != event.Action || resource != event.ResourceID {
		t.Fatal("restored Unicode changed")
	}
	// Confirm all three textual projections explicitly use the full character set.
	rows, e := other.Query(`SELECT CHARACTER_SET_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_audit' AND COLUMN_NAME IN('actor_id','action','resource_id')`)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var charset sql.NullString
		if e = rows.Scan(&charset); e != nil || !charset.Valid || charset.String != "utf8mb4" {
			t.Fatal("wrong audit charset", e)
		}
		count++
	}
	if e = rows.Err(); e != nil || count != 3 {
		t.Fatal("charset verification incomplete", e)
	}
}
