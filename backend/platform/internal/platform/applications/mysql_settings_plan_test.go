package applications

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSettingsMigrationPlanRoundTripOnSchema5(t *testing.T) {
	dsn, _, _ := schema5SettingsRoot(t)
	report, err := PlanMySQLSettings(context.Background(), dsn)
	if err != nil || !report.RoundTripVerified || report.SourceSchemaVersion != 5 || report.TargetSchemaVersion != 6 || !report.ApplySupported {
		t.Fatal(report, err)
	}
	if report.Settings != 1 {
		t.Fatal("expected settings payload", report)
	}
}

func TestSettingsMigrationPlanRejectsSchema4AndOrphanRevision(t *testing.T) {
	ctx := context.Background()
	dsn, _ := schema4SessionRoot(t)
	if _, err := PlanMySQLSettings(ctx, dsn); err == nil {
		t.Fatal("schema4 accepted")
	}
	_, root, tenant := schema5SettingsRoot(t)
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil {
		t.Fatal()
	}
	var settings map[string]json.RawMessage
	var revisions map[string]string
	json.Unmarshal(object["settings"], &settings)
	json.Unmarshal(object["setting_revisions"], &revisions)
	revisions["orphan:key"] = "ghost-rev"
	object["setting_revisions"], _ = json.Marshal(revisions)
	bad, _ := json.Marshal(object)
	if _, err := planSettingsSource(bad, tenant); err == nil {
		t.Fatal("orphan revision accepted")
	}
	_ = settings
}

func TestMySQLSettingsMigrationRehearsalRollsBack(t *testing.T) {
	ctx := context.Background()
	dsn, root, tenant := schema5SettingsRoot(t)
	db := preflightDB(t, dsn)
	if err := rehearseMySQLSettingsMigration(ctx, db, root, tenant); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_settings`).Scan(&count); err != nil || count != 0 {
		t.Fatal("rehearsal persisted rows", count, err)
	}
}

func TestSettingsMigrationPlanRejectsReservedTarget(t *testing.T) {
	dsn, _, _ := schema5SettingsRoot(t)
	db := preflightDB(t, dsn)
	if _, err := db.Exec(`CREATE TABLE cognara_fg_settings(id INT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanMySQLSettings(context.Background(), dsn); err == nil {
		t.Fatal("partial target accepted")
	}
}