package applications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// PrepareSchema8IntegrationDB upgrades a seeded MySQL validation DSN through schema 8
// with the same fixture shape used in migration and snapshot tests.
func PrepareSchema8IntegrationDB(ctx context.Context, dsn string) error {
	store, err := OpenMySQL(dsn)
	if err != nil {
		return err
	}
	store.Close()
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		return err
	}
	appPlan, err := PlanMySQLApplications(ctx, dsn)
	if err != nil {
		return fmt.Errorf("application plan: %w", err)
	}
	if _, err = ApplyMySQLApplications(ctx, dsn, appPlan.SourceSHA256); err != nil {
		return err
	}
	grantPlan, err := PlanMySQLGrants(ctx, dsn)
	if err != nil || !grantPlan.ApplySupported {
		return errors.New("grant plan failed")
	}
	if _, err = ApplyMySQLGrants(ctx, dsn, grantPlan.SourceSHA256); err != nil {
		return err
	}
	store, err = OpenMySQL(dsn)
	if err != nil {
		return err
	}
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Sessions["session-plan-1"] = Session{
			ID: "session-plan-1", AppID: "channel-replenishment", TenantID: "42", OwnerID: "demo-user",
			Title: "计划验证", CreatedAt: Now(), UpdatedAt: Now(), Revision: "rev-plan-1",
			Messages: []Message{{ID: "msg-plan-1", SessionID: "session-plan-1", Role: "user", Content: "你好", CreatedAt: Now()}},
		}
		return nil
	}); err != nil {
		store.Close()
		return err
	}
	store.Close()
	sessionPlan, err := PlanMySQLSessions(ctx, dsn)
	if err != nil || !sessionPlan.ApplySupported {
		return errors.New("session plan failed")
	}
	if _, err = ApplyMySQLSessions(ctx, dsn, sessionPlan.SourceSHA256); err != nil {
		return err
	}
	store, err = OpenMySQL(dsn)
	if err != nil {
		return err
	}
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Settings["demo-user:profile"] = json.RawMessage(`{"display_name":"计划验证"}`)
		st.SettingRevisions["demo-user:profile"] = "rev-settings-1"
		return nil
	}); err != nil {
		store.Close()
		return err
	}
	store.Close()
	settingsPlan, err := PlanMySQLSettings(ctx, dsn)
	if err != nil || !settingsPlan.ApplySupported {
		return errors.New("settings plan failed")
	}
	if _, err = ApplyMySQLSettings(ctx, dsn, settingsPlan.SourceSHA256); err != nil {
		return err
	}
	store, err = OpenMySQL(dsn)
	if err != nil {
		return err
	}
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Exceptions = append(st.Exceptions,
			map[string]any{"id": "exc-plan-1", "tenant_id": "42", "status": "OPEN", "revision": "r1", "request_id": "req-1", "occurred_at": Now()},
			map[string]any{"id": "exc-plan-2", "tenant_id": "42", "status": "ACKNOWLEDGED", "revision": "r2", "request_id": "req-2", "occurred_at": Now()},
		)
		return nil
	}); err != nil {
		store.Close()
		return err
	}
	store.Close()
	exceptionPlan, err := PlanMySQLExceptions(ctx, dsn)
	if err != nil || !exceptionPlan.ApplySupported {
		return errors.New("exception plan failed")
	}
	if _, err = ApplyMySQLExceptions(ctx, dsn, exceptionPlan.SourceSHA256); err != nil {
		return err
	}
	store, err = OpenMySQL(dsn)
	if err != nil {
		return err
	}
	if err = store.UpdateOperationalContext(ctx, func(st *State) error {
		st.Usage = append(st.Usage,
			map[string]any{"event_id": "usage-plan-1", "tenant_id": "42", "unit": "CALL", "capability": "metric", "count": 1, "occurred_at": Now()},
			map[string]any{"event_id": "usage-plan-2", "tenant_id": "42", "unit": "TOKEN", "capability": "test", "count": 2, "occurred_at": Now()},
		)
		for i := 0; i < 5; i++ {
			st.Usage = append(st.Usage, map[string]any{
				"event_id": fmt.Sprintf("usage-page-%02d", i), "tenant_id": "42", "unit": "CALL",
				"capability": "test", "count": 1, "occurred_at": "2026-11-16T00:00:00Z",
			})
		}
		return nil
	}); err != nil {
		store.Close()
		return err
	}
	store.Close()
	usagePlan, err := PlanMySQLUsage(ctx, dsn)
	if err != nil || !usagePlan.ApplySupported {
		return errors.New("usage plan failed")
	}
	if _, err = ApplyMySQLUsage(ctx, dsn, usagePlan.SourceSHA256); err != nil {
		return err
	}
	if _, err = OpenMySQL(dsn); err != nil {
		return err
	}
	return nil
}