package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func TestExceptionMigrationPlanRoundTripOnSchema6(t *testing.T) {
	dsn, _ := schema6ExceptionRoot(t)
	report, err := PlanMySQLExceptions(context.Background(), dsn)
	if err != nil || !report.RoundTripVerified || report.SourceSchemaVersion != 6 || report.TargetSchemaVersion != 7 || !report.ApplySupported {
		t.Fatal(report, err)
	}
	if report.Exceptions != 2 {
		t.Fatal("expected exception payloads", report)
	}
}

func TestExceptionMigrationPlanRejectsSchema5(t *testing.T) {
	ctx := context.Background()
	dsn, _, _ := schema5SettingsRoot(t)
	if _, err := PlanMySQLExceptions(ctx, dsn); err == nil {
		t.Fatal("schema5 accepted")
	}
}

func TestMySQLExceptionMigrationRehearsalRollsBack(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema6ExceptionRoot(t)
	db := preflightDB(t, dsn)
	if err := rehearseMySQLExceptionMigration(ctx, db, root); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_exceptions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("rehearsal persisted rows", count, err)
	}
}

func TestExceptionMigrationPlanRejectsReservedTarget(t *testing.T) {
	dsn, _ := schema6ExceptionRoot(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`CREATE TABLE cognara_fg_exceptions(id INT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanMySQLExceptions(context.Background(), dsn); err == nil {
		t.Fatal("partial target accepted")
	}
}

func TestExceptionMigrationPlanRejectsDuplicateID(t *testing.T) {
	_, root := schema6ExceptionRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	dup := []map[string]any{
		{"id": "dup", "tenant_id": "42", "status": "OPEN"},
		{"id": "dup", "tenant_id": "42", "status": "OPEN"},
	}
	object["exceptions"], _ = json.Marshal(dup)
	bad, _ := json.Marshal(object)
	if _, err := planExceptionSource(bad); err == nil {
		t.Fatal("duplicate id accepted")
	}
}