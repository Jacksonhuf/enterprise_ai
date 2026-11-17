package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// GrantMigrationPlan is a read-only snapshot report for schema3 grant split.
type GrantMigrationPlan struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	Status              string `json:"status"`
	RoundTripVerified   bool   `json:"round_trip_verified"`
	ReadOnly            bool   `json:"read_only"`
	ApplySupported      bool   `json:"apply_supported"`
	SourceSHA256        string `json:"source_sha256"`
	Grants              int    `json:"grants"`
	ActiveGrants        int    `json:"active_grants"`
	InactiveGrants      int    `json:"inactive_grants"`
	Tenants             int    `json:"tenants"`
	SampledAt           string `json:"sampled_at"`
}

// PlanMySQLGrants requires schema3 with application relational tables present.
// It does not create tables, copy data, or modify the source root.
func PlanMySQLGrants(parent context.Context, dsn string) (GrantMigrationPlan, error) {
	var report GrantMigrationPlan
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL grant plan configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL grant plan connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = validateMySQLStateSchema(ctx, db, cfg.DBName); err != nil {
		return report, errors.New("MySQL grant plan source incompatible or unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, errors.New("MySQL grant plan snapshot unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL grant plan source unavailable")
	}
	if version != 3 {
		return report, errors.New("MySQL grant plan requires schema 3; complete application migration first")
	}
	if err = validateMySQLApplicationSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL grant plan application schema incompatible")
	}
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL grant plan audit schema incompatible")
	}
	report, err = planGrantSource(raw)
	if err != nil {
		return GrantMigrationPlan{}, err
	}
	if err = verifyGrantLayout(raw); err != nil {
		return GrantMigrationPlan{}, err
	}
	report.RoundTripVerified = true
	var targets int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_grants'`).Scan(&targets); err != nil || targets != 0 {
		return GrantMigrationPlan{}, errors.New("MySQL grant plan reserved target present or unavailable; inspect before migration")
	}
	if err = tx.Commit(); err != nil {
		return GrantMigrationPlan{}, errors.New("MySQL grant plan snapshot not confirmed")
	}
	return report, nil
}

func planGrantSource(root []byte) (GrantMigrationPlan, error) {
	var result GrantMigrationPlan
	if _, err := splitGrantLayout(root); err != nil {
		return result, err
	}
	var object map[string]json.RawMessage
	var state State
	if len(root) > 64<<20 || json.Unmarshal(root, &object) != nil || object == nil || json.Unmarshal(root, &state) != nil {
		return result, errors.New("MySQL grant plan source invalid or too large")
	}
	if !canonicalFields(object, []string{"grants", "sessions", "settings", "usage", "receipts", "exceptions", "audit"}) {
		return result, errors.New("MySQL grant plan ambiguous source fields")
	}
	value, ok := object["grants"]
	if !ok {
		result.SourceSchemaVersion, result.TargetSchemaVersion = 3, 4
		result.Status, result.ReadOnly = "PLAN_VALIDATED", true
		result.ApplySupported = true
		sum := sha256.Sum256(root)
		result.SourceSHA256 = hex.EncodeToString(sum[:])
		result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
		return result, nil
	}
	if bytesTrim(value) == "null" {
		return result, errors.New("MySQL grant plan null grant container unsupported")
	}
	var items []json.RawMessage
	if json.Unmarshal(value, &items) != nil {
		return result, errors.New("MySQL grant plan grants must be an array")
	}
	tenants := map[string]bool{}
	seenIDs := map[string]bool{}
	activeTuples := map[string]string{}
	for _, raw := range items {
		var row map[string]json.RawMessage
		var g Grant
		if json.Unmarshal(raw, &row) != nil || row == nil || json.Unmarshal(raw, &g) != nil {
			return result, errors.New("MySQL grant plan grant invalid")
		}
		if !canonicalFields(row, []string{"grant_id", "tenant_id", "subject_type", "subject_id", "resource_type", "resource_id", "action", "scope", "active"}) ||
			!applicationPlanID(g.ID, 512) || !applicationPlanID(g.TenantID, 64) || !applicationPlanID(g.SubjectID, 512) || !applicationPlanID(g.ResourceID, 512) ||
			!applicationPlanID(g.SubjectType, 64) || !applicationPlanID(g.ResourceType, 64) || !applicationPlanID(g.Action, 128) || !applicationPlanID(g.Scope, 128) {
			return result, errors.New("MySQL grant plan invalid grant identity or metadata")
		}
		if strings.TrimSpace(g.SubjectType) == "" || strings.TrimSpace(g.ResourceType) == "" || strings.TrimSpace(g.Action) == "" {
			return result, errors.New("MySQL grant plan missing grant tuple fields")
		}
		if seenIDs[g.ID] {
			return result, errors.New("MySQL grant plan duplicate grant id")
		}
		seenIDs[g.ID] = true
		tenants[g.TenantID] = true
		result.Grants++
		if g.Active {
			result.ActiveGrants++
			key := strings.Join([]string{g.TenantID, g.SubjectType, g.SubjectID, g.ResourceType, g.ResourceID, g.Action, g.Scope}, "\x00")
			if other, exists := activeTuples[key]; exists {
				return result, errors.New("MySQL grant plan duplicate active grant tuple: " + other + " vs " + g.ID)
			}
			activeTuples[key] = g.ID
		} else {
			result.InactiveGrants++
		}
	}
	checksum := sha256.Sum256(root)
	result.SourceSchemaVersion, result.TargetSchemaVersion = 3, 4
	result.Status, result.ReadOnly = "PLAN_VALIDATED", true
	result.ApplySupported = true
	result.SourceSHA256 = hex.EncodeToString(checksum[:])
	result.Tenants = len(tenants)
	result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func bytesTrim(raw json.RawMessage) string {
	return strings.TrimSpace(string(raw))
}