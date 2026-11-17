package applications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func applicationPlanFixture() State {
	s := &Store{}
	s.init()
	s.seed()
	a := s.state.Apps["channel-replenishment"]
	s.state.Checks["old-check"] = Check{ID: "old-check", AppID: a.ID, Revision: "old-draft", UserID: "demo-admin", ExpiresAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return s.state
}
func TestApplicationMigrationPlanSourceAndNonBlockingHistoricalChecks(t *testing.T) {
	state := applicationPlanFixture()
	a := state.Apps["channel-replenishment"]
	b := a
	b.ID = "other-app"
	b.TenantID = "43"
	b.Releases = append([]Release(nil), a.Releases...)
	b.Releases[0].AppID = b.ID
	state.Apps[b.ID] = b
	raw, _ := json.Marshal(state)
	var object map[string]json.RawMessage
	json.Unmarshal(raw, &object)
	object["future"] = json.RawMessage(`{"preserve":true}`)
	raw, _ = json.Marshal(object)
	before := string(raw)
	report, err := planApplicationSource(raw, time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC))
	sum := sha256.Sum256(raw)
	if err != nil || report.Applications != 2 || report.Drafts != 2 || report.Releases != 2 || report.Tenants != 2 || report.Checks != 1 || report.ExpiredChecks != 1 || report.StaleChecks != 1 || !report.ReadOnly || report.ApplySupported || report.Status != "PLAN_VALIDATED" || report.SourceSHA256 != hex.EncodeToString(sum[:]) || string(raw) != before {
		t.Fatal("invalid source plan", report, err)
	}
}
func TestApplicationMigrationPlanRejectsInconsistentSource(t *testing.T) {
	tests := map[string]func(*State){
		"map-id": func(s *State) {
			a := s.Apps["channel-replenishment"]
			a.ID = "wrong"
			s.Apps["channel-replenishment"] = a
		},
		"tenant": func(s *State) {
			a := s.Apps["channel-replenishment"]
			a.TenantID = strings.Repeat("t", 65)
			s.Apps[a.ID] = a
		},
		"revision": func(s *State) { a := s.Apps["channel-replenishment"]; a.Revision = ""; s.Apps[a.ID] = a },
		"release-app": func(s *State) {
			a := s.Apps["channel-replenishment"]
			a.Releases[0].AppID = "foreign-app"
			s.Apps[a.ID] = a
		},
		"duplicate-release": func(s *State) {
			a := s.Apps["channel-replenishment"]
			a.Releases = append(a.Releases, a.Releases[0])
			s.Apps[a.ID] = a
		},
		"current-release": func(s *State) {
			a := s.Apps["channel-replenishment"]
			id := "missing"
			a.CurrentReleaseID = &id
			s.Apps[a.ID] = a
		},
		"publication-time": func(s *State) {
			a := s.Apps["channel-replenishment"]
			a.Releases[0].PublishedAt = "not-a-time"
			s.Apps[a.ID] = a
		},
		"orphan-check": func(s *State) { c := s.Checks["old-check"]; c.AppID = "foreign-app"; s.Checks[c.ID] = c },
		"check-key":    func(s *State) { c := s.Checks["old-check"]; c.ID = "wrong"; s.Checks["old-check"] = c },
		"check-time":   func(s *State) { c := s.Checks["old-check"]; c.ExpiresAt = time.Time{}; s.Checks[c.ID] = c },
		"inline-audit": func(s *State) { s.Audit = []Event{{ID: "inline"}} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			s := applicationPlanFixture()
			mutate(&s)
			raw, _ := json.Marshal(s)
			if _, err := planApplicationSource(raw, time.Now()); err == nil {
				t.Fatal("invalid source accepted")
			}
		})
	}
	for _, raw := range []string{`null`, `[]`, `{}`, `{"apps":null}`, `{"Apps":{},"apps":{}}`, `{"apps":{},"Checks":{}}`, `{"apps":{},"checks":[],"audit":[]}`} {
		if _, err := planApplicationSource([]byte(raw), time.Now()); err == nil {
			t.Fatal("invalid JSON envelope accepted", raw)
		}
	}
	// Canonical projected fields cannot be shadowed by case aliases; opaque config is retained.
	s := applicationPlanFixture()
	raw, _ := json.Marshal(s)
	for _, pair := range [][2]string{{`"tenant_id":`, `"TENANT_ID":`}, {`"release_id":`, `"RELEASE_ID":`}, {`"check_id":`, `"CHECK_ID":`}, {`"draft":{`, `"draft":null,"ignored":{`}, {`"config_snapshot":{`, `"config_snapshot":null,"ignored":{`}} {
		changed := strings.Replace(string(raw), pair[0], pair[1], 1)
		if _, err := planApplicationSource([]byte(changed), time.Now()); err == nil {
			t.Fatal("ambiguous or null projected field accepted", pair[0])
		}
	}
}
func TestMySQLApplicationPlanReadOnlyPrerequisitesAndTargets(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	db := preflightDB(t, dsn)
	if _, err := PlanMySQLApplications(context.Background(), dsn); err == nil {
		t.Fatal("empty source accepted")
	}
	if auditTableCount(t, db) != 0 {
		t.Fatal("plan initialized schema")
	}
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err = PlanMySQLApplications(context.Background(), dsn); err == nil {
		t.Fatal("schema 1 accepted without explicit audit migration")
	}
	if err = MigrateMySQLAudit(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	var before, after string
	var version int
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	report, err := PlanMySQLApplications(context.Background(), dsn)
	if err != nil || !report.RoundTripVerified || report.Applications != 1 || report.Releases != 1 || report.SourceSchemaVersion != 2 || report.TargetSchemaVersion != 3 || report.ApplySupported {
		t.Fatal(report, err)
	}
	if err = db.QueryRow(`SELECT body,schema_version FROM cognara_fg_state WHERE id=1`).Scan(&after, &version); err != nil {
		t.Fatal(err)
	}
	if before != after || version != 2 || auditTableCount(t, db) != 2 {
		t.Fatal("plan changed source")
	}
	for _, table := range []string{"cognara_fg_applications", "cognara_fg_application_drafts", "cognara_fg_application_releases", "cognara_fg_application_checks"} {
		if _, err = db.Exec("CREATE TABLE " + table + "(id INT PRIMARY KEY)"); err != nil {
			t.Fatal(err)
		}
		if _, err = PlanMySQLApplications(context.Background(), dsn); err == nil {
			t.Fatal("reserved target accepted", table)
		}
		if _, err = db.Exec("DROP TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.apps."channel-replenishment".current_release_id','missing') WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = PlanMySQLApplications(context.Background(), dsn); err == nil {
		t.Fatal("invalid persisted current release accepted")
	}
	if _, err = db.Exec(`UPDATE cognara_fg_state SET body=? WHERE id=1`, before); err != nil {
		t.Fatal(err)
	}
	if err = store.Update(func(st *State) error {
		st.Audit = append(st.Audit, Event{TenantID: "42", ID: "audit", OccurredAt: Now()})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE cognara_fg_audit SET payload=JSON_SET(payload,'$.tenant_id','foreign')`); err != nil {
		t.Fatal(err)
	}
	if _, err = PlanMySQLApplications(context.Background(), dsn); err == nil {
		t.Fatal("corrupt audit stream accepted")
	}
	if _, err = db.Exec(`UPDATE cognara_fg_state SET schema_version=99 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err = PlanMySQLApplications(context.Background(), dsn); err == nil {
		t.Fatal("future schema accepted")
	}
}
func TestMySQLApplicationPlanDoesNotTakeWriterLock(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	db := preflightDB(t, dsn)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = MigrateMySQLAudit(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future','uncommitted') WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	report, err := PlanMySQLApplications(ctx, dsn)
	if err != nil || report.Applications != 1 {
		t.Fatal("read-only plan waited for writer", err)
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if _, err = PlanMySQLApplications(canceled, dsn); err == nil {
		t.Fatal("canceled plan succeeded")
	}
}
func TestMySQLApplicationPlanCompiledCLI(t *testing.T) {
	binary := os.Getenv("COGNARA_TEST_APPLICATION_PLAN_BINARY")
	if binary == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("compiled application planner required")
		}
		t.Skip("set COGNARA_TEST_APPLICATION_PLAN_BINARY")
	}
	dsn := testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = MigrateMySQLAudit(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPLICATION_PLAN_TEST_DSN", dsn)
	command := exec.Command(binary, "--check", "--mysql-dsn-env", "APPLICATION_PLAN_TEST_DSN")
	raw, err := command.CombinedOutput()
	var report ApplicationMigrationPlan
	if err != nil || json.Unmarshal(raw, &report) != nil || report.Status != "PLAN_VALIDATED" || !report.RoundTripVerified || !report.ReadOnly || report.ApplySupported {
		t.Fatal("compiled planner failed", err, string(raw))
	}
}
