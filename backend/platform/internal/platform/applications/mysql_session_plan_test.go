package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSessionMigrationPlanRoundTripOnSchema4(t *testing.T) {
	dsn, _ := schema4SessionRoot(t)
	report, err := PlanMySQLSessions(context.Background(), dsn)
	if err != nil || !report.RoundTripVerified || report.SourceSchemaVersion != 4 || report.TargetSchemaVersion != 5 || report.ApplySupported {
		t.Fatal(report, err)
	}
	if report.Sessions != 1 || report.Messages != 1 || report.Tenants < 1 {
		t.Fatal("expected session payload", report)
	}
}

func TestSessionMigrationPlanRejectsSchema3AndDuplicateMessage(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema3GrantFixture(t)
	if _, err := PlanMySQLSessions(ctx, dsn); err == nil {
		t.Fatal("schema3 accepted")
	}
	_, root := schema4SessionRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	var sessions map[string]json.RawMessage
	if json.Unmarshal(object["sessions"], &sessions) != nil {
		t.Fatal()
	}
	raw := sessions["session-plan-1"]
	var s Session
	if json.Unmarshal(raw, &s) != nil {
		t.Fatal()
	}
	dup := append(s.Messages, s.Messages[0])
	s.Messages = dup
	sessions["session-plan-1"], _ = json.Marshal(s)
	object["sessions"], _ = json.Marshal(sessions)
	bad, _ := json.Marshal(object)
	if _, err := planSessionSource(bad); err == nil {
		t.Fatal("duplicate message id accepted")
	}
}

func TestMySQLSessionMigrationRehearsalRollsBack(t *testing.T) {
	ctx := context.Background()
	dsn, root := schema4SessionRoot(t)
	db := preflightDB(t, dsn)
	if err := rehearseMySQLSessionMigration(ctx, db, root); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_sessions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("rehearsal persisted rows", count, err)
	}
}

func TestSessionMigrationPlanRejectsReservedTarget(t *testing.T) {
	dsn, _ := schema4SessionRoot(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`CREATE TABLE cognara_fg_sessions(id INT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanMySQLSessions(context.Background(), dsn); err == nil {
		t.Fatal("partial target accepted")
	}
}