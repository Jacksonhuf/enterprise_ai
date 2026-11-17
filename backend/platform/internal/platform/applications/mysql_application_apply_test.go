package applications

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestApplyMySQLApplicationsCommitsSchema3(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	db := preflightDB(t, dsn)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	var before string
	if err = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMySQLApplications(ctx, dsn)
	if err != nil || !plan.RoundTripVerified || plan.SourceSchemaVersion != 2 {
		t.Fatal(plan, err)
	}
	if _, err = ApplyMySQLApplications(ctx, dsn, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong checksum accepted")
	}
	applied, err := ApplyMySQLApplications(ctx, dsn, plan.SourceSHA256)
	if err != nil || applied.Status != "APPLIED" || applied.ReadOnly {
		t.Fatal(applied, err)
	}
	var version int
	var after string
	if err = db.QueryRow(`SELECT body,schema_version FROM cognara_fg_state WHERE id=1`).Scan(&after, &version); err != nil || version != 3 {
		t.Fatal("version not switched", version, err)
	}
	if strings.Contains(after, `"apps"`) || strings.Contains(after, `"checks"`) {
		t.Fatal("aggregate apps/checks still in root body")
	}
	for _, table := range []string{"cognara_fg_applications", "cognara_fg_application_drafts", "cognara_fg_application_releases", "cognara_fg_application_envelope"} {
		var count int
		if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count == 0 {
			t.Fatal("relational data missing", table, count, err)
		}
	}
	if plan.Checks > 0 {
		var checks int
		if err = db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_application_checks`).Scan(&checks); err != nil || checks != plan.Checks {
			t.Fatal("checks not migrated", checks, plan.Checks, err)
		}
	}
	runtime, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal("runtime rejected applied schema3", err)
	}
	defer runtime.Close()
	st, err := runtime.readSQLContext(ctx)
	if err != nil || len(st.Apps) != plan.Applications {
		t.Fatal("runtime state incomplete", err, len(st.Apps))
	}
	snapshot := filepath.Join(t.TempDir(), "post-apply.json")
	if _, err = ExportMySQLSnapshot(ctx, dsn, snapshot); err != nil {
		t.Fatal("post-apply export failed", err)
	}
	if _, err = ApplyMySQLApplications(ctx, dsn, plan.SourceSHA256); err == nil {
		t.Fatal("repeat apply accepted")
	}
	if before == after {
		t.Fatal("source body unchanged unexpectedly")
	}
}

func TestApplyMySQLApplicationsRejectsPrerequisiteFailures(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	if _, err := ApplyMySQLApplications(ctx, dsn, strings.Repeat("a", 64)); err == nil {
		t.Fatal("empty schema accepted")
	}
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	store.Close()
	plan, err := PlanMySQLApplications(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	wrong := plan.SourceSHA256
	if len(wrong) == 64 && wrong[63] != 'a' {
		wrong = wrong[:63] + "a"
	} else if len(wrong) == 64 {
		wrong = wrong[:63] + "b"
	} else {
		wrong = strings.Repeat("0", 64)
	}
	if _, err = ApplyMySQLApplications(ctx, dsn, wrong); err == nil {
		t.Fatal("checksum drift accepted")
	}
}

func TestOpenMySQLRejectsHandTunedSchema3(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	db := preflightDB(t, dsn)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	store.Close()
	if _, err = db.Exec(`UPDATE cognara_fg_state SET schema_version=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, e := OpenMySQL(dsn); e == nil {
		t.Fatal("hand-tuned schema3 accepted")
	}
}