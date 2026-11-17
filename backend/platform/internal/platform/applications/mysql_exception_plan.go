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

// ExceptionMigrationPlan is a read-only snapshot report for schema6 exception split.
type ExceptionMigrationPlan struct {
	SourceSchemaVersion int    `json:"source_schema_version"`
	TargetSchemaVersion int    `json:"target_schema_version"`
	Status              string `json:"status"`
	RoundTripVerified   bool   `json:"round_trip_verified"`
	ReadOnly            bool   `json:"read_only"`
	ApplySupported      bool   `json:"apply_supported"`
	SourceSHA256        string `json:"source_sha256"`
	Exceptions          int    `json:"exceptions"`
	Tenants             int    `json:"tenants"`
	SampledAt           string `json:"sampled_at"`
}

// PlanMySQLExceptions requires schema6 with application, grant, session, and settings relational tables present.
func PlanMySQLExceptions(parent context.Context, dsn string) (ExceptionMigrationPlan, error) {
	var report ExceptionMigrationPlan
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL exception plan configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL exception plan connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = validateMySQLStateSchema(ctx, db, cfg.DBName); err != nil {
		return report, errors.New("MySQL exception plan source incompatible or unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return report, errors.New("MySQL exception plan snapshot unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL exception plan source unavailable")
	}
	if version != 6 {
		return report, errors.New("MySQL exception plan requires schema 6; complete settings migration first")
	}
	if err = validateMySQLApplicationSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL exception plan application schema incompatible")
	}
	if err = validateMySQLGrantSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL exception plan grant schema incompatible")
	}
	if err = validateMySQLSessionSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL exception plan session schema incompatible")
	}
	if err = validateMySQLSettingsSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL exception plan settings schema incompatible")
	}
	if err = validateMySQLAuditSchema(ctx, tx); err != nil {
		return report, errors.New("MySQL exception plan audit schema incompatible")
	}
	if _, _, err = readMySQLApplicationV6(ctx, tx, false); err != nil {
		return report, errors.New("MySQL exception plan relational read failed")
	}
	report, err = planExceptionSource(raw)
	if err != nil {
		return ExceptionMigrationPlan{}, err
	}
	if err = verifyExceptionLayout(raw); err != nil {
		return ExceptionMigrationPlan{}, err
	}
	report.RoundTripVerified = true
	var targets int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='cognara_fg_exceptions'`).Scan(&targets); err != nil || targets != 0 {
		return ExceptionMigrationPlan{}, errors.New("MySQL exception plan reserved target present or unavailable; inspect before migration")
	}
	if err = tx.Commit(); err != nil {
		return ExceptionMigrationPlan{}, errors.New("MySQL exception plan snapshot not confirmed")
	}
	return report, nil
}

func planExceptionSource(root []byte) (ExceptionMigrationPlan, error) {
	var result ExceptionMigrationPlan
	layout, err := splitExceptionLayout(root)
	if err != nil {
		return result, err
	}
	var object map[string]json.RawMessage
	if len(root) > 64<<20 || json.Unmarshal(root, &object) != nil || object == nil {
		return result, errors.New("MySQL exception plan source invalid or too large")
	}
	if !canonicalFields(object, []string{"exceptions", "usage", "receipts", "audit"}) {
		return result, errors.New("MySQL exception plan ambiguous source fields")
	}
	tenants := map[string]bool{}
	seen := map[string]bool{}
	for _, row := range layout.Exceptions {
		if seen[row.ExceptionID] {
			return result, errors.New("MySQL exception plan duplicate exception id")
		}
		seen[row.ExceptionID] = true
		tenants[row.TenantID] = true
	}
	result.Exceptions = len(layout.Exceptions)
	result.Tenants = len(tenants)
	checksum := sha256.Sum256(root)
	result.SourceSchemaVersion, result.TargetSchemaVersion = 6, 7
	result.Status, result.ReadOnly = "PLAN_VALIDATED", true
	result.SourceSHA256 = hex.EncodeToString(checksum[:])
	result.SampledAt = time.Now().UTC().Format(time.RFC3339Nano)
	return result, nil
}