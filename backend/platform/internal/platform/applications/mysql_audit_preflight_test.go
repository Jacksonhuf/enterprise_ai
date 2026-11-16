package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestAuditMigrationSourceValidation(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `{"AUDIT":[]}`, `{"audit":[],"Audit":[]}`,
		`{"audit":[{"event_id":"a","occurred_at":"bad"}]}`,
		`{"audit":[{"event_id":"","occurred_at":"2026-11-16T00:00:00Z"}]}`,
		`{"audit":[{"event_id":"a","occurred_at":"2026-11-16T00:00:00Z","future":true}]}`,
		`{"audit":[{"event_id":"a","occurred_at":"2026-11-16T00:00:00Z"},{"event_id":"a","occurred_at":"2026-11-16T00:00:00Z"}]}`,
	} {
		if _, _, e := auditMigrationSource([]byte(raw)); e == nil {
			t.Fatal("invalid source accepted", raw)
		}
	}
	st, object, e := auditMigrationSource([]byte(`{"future":{"keep":true},"audit":[{"tenant_id":"42","event_id":"same","occurred_at":"0001-01-01t00:00:00z"},{"tenant_id":"43","event_id":"same","occurred_at":"2026-11-16T00:00:00.123456789Z"}]}`))
	if e != nil || len(st.Audit) != 2 || string(object["future"]) != `{"keep":true}` {
		t.Fatal("valid source rejected", e)
	}
}
func preflightDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
func auditTableCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if e := db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN('cognara_fg_state','cognara_fg_audit')`).Scan(&count); e != nil {
		t.Fatal(e)
	}
	return count
}
func TestMySQLAuditPreflightReadOnlyAndApplyAgreement(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	db := preflightDB(t, dsn)
	if _, e := CheckMySQLAuditMigration(context.Background(), dsn); e == nil {
		t.Fatal("empty source accepted")
	}
	if auditTableCount(t, db) != 0 {
		t.Fatal("check created tables")
	}
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Update(func(st *State) error {
		st.Audit = []Event{{TenantID: "42", ID: "first", OccurredAt: "2026-11-16T00:00:00.123456789Z"}, {TenantID: "42", ID: "ancient", OccurredAt: "0001-01-01t00:00:00z"}}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	var before, after string
	if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); e != nil {
		t.Fatal(e)
	}
	report, e := CheckMySQLAuditMigration(context.Background(), dsn)
	if e != nil || report.Status != "READY" || report.AuditEvents != 2 || !report.ReadOnly || report.SourceSchemaVersion != 1 || report.TargetSchemaVersion != 2 {
		t.Fatal("preflight", report, e)
	}
	if auditTableCount(t, db) != 1 {
		t.Fatal("check created audit target")
	}
	if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); e != nil || before != after {
		t.Fatal("check altered source", e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	report, e = CheckMySQLAuditMigration(context.Background(), dsn)
	if e != nil || report.Status != "ALREADY_APPLIED" || report.SourceSchemaVersion != 2 || report.AuditEvents != 2 {
		t.Fatal("v2 verification", report, e)
	}
	// Preserve unknown imported payload fields and accepted case aliases in v2 checks.
	if _, e = db.Exec(`UPDATE cognara_fg_audit SET payload=JSON_SET(payload,'$.Outcome','success','$.future',true) WHERE sequence_no=1`); e != nil {
		t.Fatal(e)
	}
	if _, e = CheckMySQLAuditMigration(context.Background(), dsn); e != nil {
		t.Fatal("accepted v2 aliases rejected", e)
	}
}
func TestMySQLAuditPreflightRejectsBadSourceAndTargets(t *testing.T) {
	for _, scenario := range []string{"duplicate", "alias", "unknown-audit-field", "wrong-target", "nonempty-target", "sequence-gap", "identity-mismatch", "invalid-time"} {
		t.Run(scenario, func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			db := preflightDB(t, dsn)
			s, e := OpenMySQL(dsn)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			event := Event{TenantID: "42", ID: "event", OccurredAt: "2026-11-16T00:00:00Z"}
			if e = s.Update(func(st *State) error {
				st.Audit = []Event{event}
				if scenario == "duplicate" {
					st.Audit = append(st.Audit, event)
				}
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			switch scenario {
			case "alias":
				_, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.Audit',JSON_ARRAY()) WHERE id=1`)
			case "unknown-audit-field":
				_, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.audit[0].future',true) WHERE id=1`)
			case "wrong-target":
				_, e = db.Exec(`CREATE TABLE cognara_fg_audit(id INT PRIMARY KEY) ENGINE=InnoDB`)
			case "nonempty-target":
				_, e = db.Exec(mysqlAuditDDL)
				if e == nil {
					raw, _ := json.Marshal(event)
					_, e = db.Exec(`INSERT INTO cognara_fg_audit VALUES(?,?,1,NULL,'','','',?)`, event.TenantID, event.ID, string(raw))
				}
			case "sequence-gap", "identity-mismatch", "invalid-time":
				e = MigrateMySQLAudit(context.Background(), dsn)
				if e == nil {
					switch scenario {
					case "sequence-gap":
						_, e = db.Exec(`UPDATE cognara_fg_audit SET sequence_no=2`)
					case "identity-mismatch":
						_, e = db.Exec(`UPDATE cognara_fg_audit SET payload=JSON_SET(payload,'$.event_id','different')`)
					case "invalid-time":
						_, e = db.Exec(`UPDATE cognara_fg_audit SET payload=JSON_SET(payload,'$.occurred_at','bad')`)
					}
				}
			}
			if e != nil {
				t.Fatal(e)
			}
			var before, after string
			var versionBefore, versionAfter int
			if e = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&versionBefore, &before); e != nil {
				t.Fatal(e)
			}
			if _, e = CheckMySQLAuditMigration(context.Background(), dsn); e == nil {
				t.Fatal("invalid preflight accepted")
			}
			if e = db.QueryRow(`SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&versionAfter, &after); e != nil || before != after || versionBefore != versionAfter {
				t.Fatal("rejected check changed state", e)
			}
			if versionBefore == 1 {
				if e = MigrateMySQLAudit(context.Background(), dsn); e == nil {
					t.Fatal("apply disagrees with preflight")
				}
			}
		})
	}
}
func TestMySQLAuditPreflightDoesNotTakeStateWriteLock(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	db := preflightDB(t, dsn)
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow(`SELECT schema_version FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, e = CheckMySQLAuditMigration(ctx, dsn); e != nil {
		t.Fatal("read-only check waited on writer", e)
	}
}
func TestMySQLAuditMigrationCompiledCheck(t *testing.T) {
	binary := os.Getenv("COGNARA_TEST_MIGRATE_BINARY")
	if binary == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("compiled migration binary required")
		}
		t.Skip("set compiled migration binary")
	}
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	db := preflightDB(t, dsn)
	run := func(args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env = append(os.Environ(), "COGNARA_PREFLIGHT_DSN="+dsn)
		return cmd.CombinedOutput()
	}
	for _, args := range [][]string{{}, {"--check", "--apply"}, {"--check", "extra"}} {
		if _, e = run(args...); e == nil {
			t.Fatal("ambiguous flags accepted")
		}
	}
	raw, e := run("--check", "--mysql-dsn-env", "COGNARA_PREFLIGHT_DSN")
	var report AuditMigrationReport
	if e != nil || json.Unmarshal(raw, &report) != nil || report.Status != "READY" || !report.ReadOnly {
		t.Fatal("compiled check", string(raw), e)
	}
	if auditTableCount(t, db) != 1 {
		t.Fatal("compiled check wrote schema")
	}
	if _, e = run("--apply", "--mysql-dsn-env", "COGNARA_PREFLIGHT_DSN"); e != nil {
		t.Fatal(e)
	}
	raw, e = run("--check", "--mysql-dsn-env", "COGNARA_PREFLIGHT_DSN")
	if e != nil || json.Unmarshal(raw, &report) != nil || report.Status != "ALREADY_APPLIED" {
		t.Fatal("compiled v2 check", string(raw), e)
	}
	raw, e = run("--check", "--mysql-dsn-env", "COGNARA_PREFLIGHT_MISSING_ENV")
	if e == nil || strings.Contains(string(raw), dsn) {
		t.Fatal("invalid DSN handling")
	}
}
