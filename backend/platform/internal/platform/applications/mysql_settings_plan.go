package applications

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
)

// SettingsMigrationPlan is a read-only snapshot report for schema5 settings split.
type SettingsMigrationPlan struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	Status              string `json:"status"`
	RoundTripVerified   bool   `json:"round_trip_verified"`
	ReadOnly            bool   `json:"read_only"`
	ApplySupported      bool   `json:"apply_supported"`
	SourceSHA256        string `json:"source_sha256"`
	Settings            int    `json:"settings"`
	Tenants             int    `json:"tenants"`
	SampledAt           string `json:"sampled_at"`
}

// PlanMySQLSettings requires schema5 with application, grant, and session relational tables present.
func PlanMySQLSettings(parent context.Context, dsn string) (SettingsMigrationPlan, error) {
	var report SettingsMigrationPlan
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL settings plan configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL settings plan connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = validateMySQLStateSchema(ctx, db, cfg.DBName); err != nil {
		return report, errors.New("MySQL settings plan source incompatible or unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, errors.New("MySQL settings plan snapshot unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL settings plan source unavailable")
	}
	if version != 5 {
		return report, errors.New("MySQL settings plan requires schema 5; complete session migration first")
	}
	if err = validateMySQLApplicationSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL settings plan application schema incompatible")
	}
	if err = validateMySQLGrantSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL settings plan grant schema incompatible")
	}
	if err = validateMySQLSessionSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL settings plan session schema incompatible")
	}
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL settings plan audit schema incompatible")
	}
	st, _, err := readMySQLApplicationV5(ctx, tx, false)
	if err != nil {
		return report, errors.New("MySQL settings plan relational read failed")
	}
	tenant, err := dominantTenantID(st)
	if err != nil {
		return report, err
	}
	report, err = planSettingsSource(raw, tenant)
	if err != nil {
		return SettingsMigrationPlan{}, err
	}
	if err = verifySettingsLayout(raw, tenant); err != nil {
		return SettingsMigrationPlan{}, err
	}
	report.RoundTripVerified = true
	var targets int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_settings'`).Scan(&targets); err != nil || targets != 0 {
		return SettingsMigrationPlan{}, errors.New("MySQL settings plan reserved target present or unavailable; inspect before migration")
	}
	if err = tx.Commit(); err != nil {
		return SettingsMigrationPlan{}, errors.New("MySQL settings plan snapshot not confirmed")
	}
	return report, nil
}

func planSettingsSource(root []byte, tenant string) (SettingsMigrationPlan, error) {
	var result SettingsMigrationPlan
	layout, err := splitSettingsLayout(root)
	if err != nil {
		return result, err
	}
	layout.InferredTenant = tenant
	for i := range layout.Settings {
		layout.Settings[i].TenantID = tenant
	}
	var object map[string]json.RawMessage
	if len(root) > 64<<20 || json.Unmarshal(root, &object) != nil || object == nil {
		return result, errors.New("MySQL settings plan source invalid or too large")
	}
	if !canonicalFields(object, []string{"settings", "setting_revisions", "usage", "receipts", "exceptions", "audit"}) {
		return result, errors.New("MySQL settings plan ambiguous source fields")
	}
	result.Settings = len(layout.Settings)
	if result.Settings > 0 && tenant == "" {
		return result, errors.New("MySQL settings plan tenant required")
	}
	checksum := sha256.Sum256(root)
	result.SourceSchemaVersion, result.TargetSchemaVersion = 5, 6
	result.Status, result.ReadOnly = "PLAN_VALIDATED", true
	result.SourceSHA256 = hex.EncodeToString(checksum[:])
	if result.Settings > 0 {
		result.Tenants = 1
	}
	result.ApplySupported = true
	result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}