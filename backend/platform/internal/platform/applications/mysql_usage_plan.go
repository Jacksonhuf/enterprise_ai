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

// UsageMigrationPlan is a read-only snapshot report for schema7 usage split.
type UsageMigrationPlan struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	Status              string `json:"status"`
	RoundTripVerified   bool   `json:"round_trip_verified"`
	ReadOnly            bool   `json:"read_only"`
	ApplySupported      bool   `json:"apply_supported"`
	SourceSHA256        string `json:"source_sha256"`
	Usage               int    `json:"usage"`
	Tenants             int    `json:"tenants"`
	SampledAt           string `json:"sampled_at"`
}

// PlanMySQLUsage requires schema7 with application, grant, session, settings, and exception relational tables present.
func PlanMySQLUsage(parent context.Context, dsn string) (UsageMigrationPlan, error) {
	var report UsageMigrationPlan
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL usage plan configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL usage plan connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = validateMySQLStateSchema(ctx, db, cfg.DBName); err != nil {
		return report, errors.New("MySQL usage plan source incompatible or unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, errors.New("MySQL usage plan snapshot unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL usage plan source unavailable")
	}
	if version != 7 {
		return report, errors.New("MySQL usage plan requires schema 7; complete exception migration first")
	}
	if err = validateMySQLApplicationSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL usage plan application schema incompatible")
	}
	if err = validateMySQLGrantSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL usage plan grant schema incompatible")
	}
	if err = validateMySQLSessionSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL usage plan session schema incompatible")
	}
	if err = validateMySQLSettingsSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL usage plan settings schema incompatible")
	}
	if err = validateMySQLExceptionSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL usage plan exception schema incompatible")
	}
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL usage plan audit schema incompatible")
	}
	if _, _, err = readMySQLApplicationV7(ctx, tx, false); err != nil {
		return report, errors.New("MySQL usage plan relational read failed")
	}
	report, err = planUsageSource(raw)
	if err != nil {
		return UsageMigrationPlan{}, err
	}
	if err = verifyUsageLayout(raw); err != nil {
		return UsageMigrationPlan{}, err
	}
	report.RoundTripVerified = true
	var targets int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_usage'`).Scan(&targets); err != nil || targets != 0 {
		return UsageMigrationPlan{}, errors.New("MySQL usage plan reserved target present or unavailable; inspect before migration")
	}
	if err = tx.Commit(); err != nil {
		return UsageMigrationPlan{}, errors.New("MySQL usage plan snapshot not confirmed")
	}
	return report, nil
}

func planUsageSource(root []byte) (UsageMigrationPlan, error) {
	var result UsageMigrationPlan
	layout, err := splitUsageLayout(root)
	if err != nil {
		return result, err
	}
	var object map[string]json.RawMessage
	if len(root) > 64<<20 || json.Unmarshal(root, &object) != nil || object == nil {
		return result, errors.New("MySQL usage plan source invalid or too large")
	}
	if !canonicalFields(object, []string{"usage", "receipts", "audit"}) {
		return result, errors.New("MySQL usage plan ambiguous source fields")
	}
	tenants := map[string]bool{}
	seen := map[string]bool{}
	for _, row := range layout.Usage {
		if seen[row.EventID] {
			return result, errors.New("MySQL usage plan duplicate event id")
		}
		seen[row.EventID] = true
		tenants[row.TenantID] = true
	}
	result.Usage = len(layout.Usage)
	result.Tenants = len(tenants)
	checksum := sha256.Sum256(root)
	result.SourceSchemaVersion, result.TargetSchemaVersion = 7, 8
	result.Status, result.ReadOnly = "PLAN_VALIDATED", true
	result.ApplySupported = true
	result.SourceSHA256 = hex.EncodeToString(checksum[:])
	result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}