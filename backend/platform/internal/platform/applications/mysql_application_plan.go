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

// ApplicationMigrationPlan is a read-only snapshot report from --check; apply uses
// SourceSHA256 with ApplyMySQLApplications or cognara-migrate-applications --apply.
type ApplicationMigrationPlan struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	Status              string `json:"status"`
	RoundTripVerified   bool   `json:"round_trip_verified"`
	ReadOnly            bool   `json:"read_only"`
	ApplySupported      bool   `json:"apply_supported"`
	SourceSHA256        string `json:"source_sha256"`
	Applications        int    `json:"applications"`
	Drafts              int    `json:"drafts"`
	Releases            int    `json:"releases"`
	Checks              int    `json:"checks"`
	ExpiredChecks       int    `json:"expired_checks"`
	StaleChecks         int    `json:"stale_checks"`
	Tenants             int    `json:"tenants"`
	AuditEvents         uint64 `json:"audit_events"`
	SampledAt           string `json:"sampled_at"`
}

// PlanMySQLApplications does not initialize, seed, create tables, lock for update,
// or modify source data. Audit migration to schema 2 is an explicit prerequisite.
func PlanMySQLApplications(parent context.Context, dsn string) (ApplicationMigrationPlan, error) {
	var report ApplicationMigrationPlan
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL application plan configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL application plan connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = validateMySQLStateSchema(ctx, db, cfg.DBName); err != nil {
		return report, errors.New("MySQL application plan source incompatible or unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, errors.New("MySQL application plan snapshot unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL application plan source unavailable")
	}
	if version != 2 {
		return report, errors.New("MySQL application plan requires schema 2; complete audit migration first; other versions unsupported")
	}
	sampled := time.Now().UTC()
	report, err = planApplicationSource(raw, sampled)
	if err != nil {
		return ApplicationMigrationPlan{}, err
	}
	if err = verifyApplicationLayout(raw, sampled); err != nil {
		return ApplicationMigrationPlan{}, err
	}
	report.RoundTripVerified = true
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return ApplicationMigrationPlan{}, errors.New("MySQL application plan audit schema incompatible")
	}
	report.AuditEvents, err = verifyMySQLAuditStream(ctx, tx)
	if err != nil {
		return ApplicationMigrationPlan{}, errors.New("MySQL application plan audit records invalid or unavailable")
	}
	// Reserved target names must be absent. Do not bless arbitrary partial DDL or
	// pre-existing data. A future apply command must inspect its own target schema.
	var targets int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN ('cognara_fg_applications','cognara_fg_application_drafts','cognara_fg_application_releases','cognara_fg_application_checks','cognara_fg_application_envelope')`).Scan(&targets); err != nil || targets != 0 {
		return ApplicationMigrationPlan{}, errors.New("MySQL application plan reserved targets present or unavailable; inspect before migration")
	}
	if err = tx.Commit(); err != nil {
		return ApplicationMigrationPlan{}, errors.New("MySQL application plan snapshot not confirmed")
	}
	return report, nil
}

func planApplicationSource(raw []byte, sampled time.Time) (ApplicationMigrationPlan, error) {
	var result ApplicationMigrationPlan
	var source map[string]json.RawMessage
	var state State
	if len(raw) > 64<<20 || json.Unmarshal(raw, &source) != nil || source == nil || json.Unmarshal(raw, &state) != nil || state.Apps == nil {
		return result, errors.New("MySQL application plan source invalid or too large")
	}
	if !canonicalFields(source, []string{"apps", "checks", "audit"}) || len(state.Audit) != 0 {
		return result, errors.New("MySQL application plan ambiguous source fields or audit storage")
	}
	tenants := map[string]bool{}
	var apps map[string]json.RawMessage
	if json.Unmarshal(source["apps"], &apps) != nil {
		return result, errors.New("MySQL application plan apps invalid")
	}
	for key, a := range state.Apps {
		var object map[string]json.RawMessage
		if json.Unmarshal(apps[key], &object) != nil || object == nil || !canonicalFields(object, []string{"app_id", "agent_id", "tenant_id", "owner_id", "draft", "draft_revision", "current_release_id", "releases", "updated_at"}) ||
			key != a.ID || !applicationPlanID(a.ID, 512) || !applicationPlanID(a.TenantID, 64) || !applicationPlanID(a.OwnerID, 512) || !applicationPlanID(a.AgentID, 512) || !applicationPlanID(a.Revision, 512) || !applicationPlanTime(a.UpdatedAt) {
			return result, errors.New("MySQL application plan invalid application identity, revision or metadata")
		}
		var draft map[string]json.RawMessage
		if json.Unmarshal(object["draft"], &draft) != nil || draft == nil {
			return result, errors.New("MySQL application plan draft must be an object")
		}
		tenants[a.TenantID] = true
		seen := map[string]bool{}
		var releases []json.RawMessage
		if json.Unmarshal(object["releases"], &releases) != nil {
			return result, errors.New("MySQL application plan releases invalid")
		}
		for i, r := range a.Releases {
			var release map[string]json.RawMessage
			if json.Unmarshal(releases[i], &release) != nil || release == nil || !canonicalFields(release, []string{"release_id", "app_id", "config_snapshot", "release_note", "published_at", "published_by"}) ||
				!applicationPlanID(r.ID, 512) || r.AppID != a.ID || seen[r.ID] || !applicationPlanID(r.PublishedBy, 512) || !applicationPlanTime(r.PublishedAt) {
				return result, errors.New("MySQL application plan invalid or duplicate release")
			}
			var config map[string]json.RawMessage
			if json.Unmarshal(release["config_snapshot"], &config) != nil || config == nil {
				return result, errors.New("MySQL application plan release snapshot must be an object")
			}
			seen[r.ID] = true
			result.Releases++
		}
		if a.CurrentReleaseID != nil && !seen[*a.CurrentReleaseID] {
			return result, errors.New("MySQL application plan current release is missing from its application")
		}
	}
	var checks map[string]json.RawMessage
	if value, ok := source["checks"]; ok && json.Unmarshal(value, &checks) != nil {
		return result, errors.New("MySQL application plan checks invalid")
	}
	for key, c := range state.Checks {
		var object map[string]json.RawMessage
		a, exists := state.Apps[c.AppID]
		if json.Unmarshal(checks[key], &object) != nil || object == nil || !canonicalFields(object, []string{"check_id", "app_id", "draft_revision", "user_id", "expires_at", "validation_id"}) || !exists || key != c.ID ||
			!applicationPlanID(c.ID, 512) || !applicationPlanID(c.UserID, 512) || !applicationPlanID(c.Revision, 512) || c.ExpiresAt.IsZero() || len(c.ValidationID) > 512 {
			return result, errors.New("MySQL application plan invalid or orphaned release check")
		}
		if !c.ExpiresAt.After(sampled) {
			result.ExpiredChecks++
		}
		if c.Revision != a.Revision {
			result.StaleChecks++
		}
	}
	checksum := sha256.Sum256(raw)
	result.SourceSchemaVersion, result.TargetSchemaVersion = 2, 3
	result.Status, result.ReadOnly = "PLAN_VALIDATED", true
	result.SourceSHA256 = hex.EncodeToString(checksum[:])
	result.Applications, result.Drafts, result.Checks, result.Tenants = len(state.Apps), len(state.Apps), len(state.Checks), len(tenants)
	result.SampledAt = sampled.UTC().Format(time.RFC3339Nano)
	return result, nil
}

// Unknown opaque fields are retained by the future raw-envelope migration;
// aliases for projected fields are rejected rather than silently choosing one.
func canonicalFields(object map[string]json.RawMessage, fields []string) bool {
	for key := range object {
		for _, field := range fields {
			if key != field && strings.EqualFold(key, field) {
				return false
			}
		}
	}
	return true
}
func applicationPlanID(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= limit
}
func applicationPlanTime(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, strings.ToUpper(value))
	return len(value) <= 64 && err == nil
}
