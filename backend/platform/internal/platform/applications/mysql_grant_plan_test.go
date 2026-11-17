package applications

import (
	"context"
	"encoding/json"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
)

func schema3GrantFixture(t *testing.T) (dsn string, root []byte) {
	t.Helper()
	ctx := context.Background()
	dsn = testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMySQLApplications(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyMySQLApplications(ctx, dsn, plan.SourceSHA256); err != nil {
		t.Fatal(err)
	}
	if err = preflightDB(t, dsn).QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		t.Fatal(err)
	}
	return dsn, root
}

func TestGrantMigrationPlanRoundTripOnSchema3(t *testing.T) {
	dsn, _ := schema3GrantFixture(t)
	report, err := PlanMySQLGrants(context.Background(), dsn)
	if err != nil || !report.RoundTripVerified || report.SourceSchemaVersion != 3 || report.TargetSchemaVersion != 4 || !report.ApplySupported {
		t.Fatal(report, err)
	}
	if report.Grants < 1 || report.ActiveGrants < 1 || report.Tenants < 1 {
		t.Fatal("expected seed grants", report)
	}
}

func TestGrantMigrationPlanRejectsSchema2AndDuplicateActiveTuple(t *testing.T) {
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	store, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	if _, err = PlanMySQLGrants(ctx, dsn); err == nil {
		t.Fatal("schema2 accepted")
	}
	_, root := schema3GrantFixture(t)
	var object map[string]json.RawMessage
	json.Unmarshal(root, &object)
	var grants []json.RawMessage
	json.Unmarshal(object["grants"], &grants)
	duplicate, _ := json.Marshal(append(grants, grants[0]))
	object["grants"] = duplicate
	bad, _ := json.Marshal(object)
	if _, err = planGrantSource(bad); err == nil {
		t.Fatal("duplicate grant id accepted")
	}
	dupActive := grants[0]
	var g Grant
	json.Unmarshal(dupActive, &g)
	g.ID = "grant-dup-active"
	dupActive, _ = json.Marshal(g)
	duplicate, _ = json.Marshal(append(grants, dupActive))
	object["grants"] = duplicate
	bad, _ = json.Marshal(object)
	if _, err = planGrantSource(bad); err == nil {
		t.Fatal("duplicate active tuple accepted")
	}
}

func TestMySQLGrantMigrationRehearsalRollsBack(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema3GrantFixture(t)
	db := preflightDB(t, dsn)
	if err := rehearseMySQLGrantMigration(ctx, db, root); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_grants`).Scan(&count); err != nil || count != 0 {
		t.Fatal("rehearsal persisted rows", count, err)
	}
}

func TestGrantMigrationPlanRejectsReservedTarget(t *testing.T) {
	dsn, _ := schema3GrantFixture(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`CREATE TABLE cognara_fg_grants(id INT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanMySQLGrants(context.Background(), dsn); err == nil {
		t.Fatal("partial target accepted")
	}
}