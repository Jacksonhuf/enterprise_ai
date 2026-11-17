package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func TestUsageMigrationPlanRoundTripOnSchema7(t *testing.T) {
	dsn, _ := schema7UsageRoot(t)
	report, err := PlanMySQLUsage(context.Background(), dsn)
	if err != nil || !report.RoundTripVerified || report.SourceSchemaVersion != 7 || report.TargetSchemaVersion != 8 || !report.ApplySupported {
		t.Fatal(report, err)
	}
	if report.Usage != 2 {
		t.Fatal("expected usage payloads", report)
	}
}

func TestUsageMigrationPlanRejectsSchema6(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema6ExceptionRoot(t)
	if _, err := PlanMySQLUsage(ctx, dsn); err == nil {
		t.Fatal("schema6 accepted")
	}
}

func TestMySQLUsageMigrationRehearsalRollsBack(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema7UsageRoot(t)
	db := preflightDB(t, dsn)
	if err := rehearseMySQLUsageMigration(ctx, db, root); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_usage`).Scan(&count); err != nil || count != 0 {
		t.Fatal("rehearsal persisted rows", count, err)
	}
}

func TestUsageMigrationPlanRejectsReservedTarget(t *testing.T) {
	dsn, _ := schema7UsageRoot(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`CREATE TABLE cognara_fg_usage(id INT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanMySQLUsage(context.Background(), dsn); err == nil {
		t.Fatal("partial target accepted")
	}
}

func TestUsageMigrationPlanRejectsDuplicateEventID(t *testing.T) {
	_, root := schema7UsageRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	dup := []map[string]any{
		{"event_id": "dup", "tenant_id": "42", "unit": "CALL"},
		{"event_id": "dup", "tenant_id": "42", "unit": "TOKEN"},
	}
	object["usage"], _ = json.Marshal(dup)
	bad, _ := json.Marshal(object)
	if _, err := planUsageSource(bad); err == nil {
		t.Fatal("duplicate id accepted")
	}
}