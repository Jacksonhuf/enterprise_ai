package applications

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ApplyMySQLExceptions migrates exceptions out of schema6 state.body into cognara_fg_exceptions
// and bumps schema_version to 7 atomically.
func ApplyMySQLExceptions(parent context.Context, dsn, expectedSHA string) (ExceptionMigrationPlan, error) {
	var report ExceptionMigrationPlan
	expected, err := hex.DecodeString(strings.TrimSpace(expectedSHA))
	if err != nil || len(expected) != sha256.Size {
		return report, errors.New("expected source SHA-256 is required")
	}
	plan, err := PlanMySQLExceptions(parent, dsn)
	if err != nil {
		return report, err
	}
	if !plan.RoundTripVerified || !plan.ApplySupported {
		return report, errors.New("exception migration plan failed")
	}
	if !bytes.Equal(expected, mustDecodeHex(plan.SourceSHA256)) {
		return report, errors.New("exception migration source checksum mismatch")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL exception apply configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 30*time.Second, 30*time.Second, 30*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL exception apply connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = createMySQLExceptionSchema(ctx, db); err != nil {
		return report, errors.New("MySQL exception apply schema unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return report, errors.New("MySQL exception apply transaction unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL exception apply source unavailable")
	}
	if version != 6 {
		return report, errors.New("exception migration requires schema 6")
	}
	sum := sha256.Sum256(raw)
	if !bytes.Equal(sum[:], expected) {
		return report, errors.New("exception migration source changed during apply")
	}
	if err = stageMySQLExceptionRoot(ctx, tx); err != nil {
		return report, errors.New("exception root handoff failed")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET schema_version=7 WHERE id=1`); err != nil {
		return report, errors.New("exception schema version switch failed")
	}
	if _, _, err = readMySQLApplicationV7(ctx, tx, true); err != nil {
		return report, errors.New("exception migration reconciliation failed")
	}
	if err = tx.Commit(); err != nil {
		return report, errors.New("exception migration commit not confirmed; inspect before retry")
	}
	plan.Status, plan.ReadOnly, plan.ApplySupported = "APPLIED", false, true
	return plan, nil
}