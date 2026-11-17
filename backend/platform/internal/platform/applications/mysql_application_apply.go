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

// ApplyMySQLApplications performs the schema2→schema3 application split in one
// commit: relational rows are populated from the locked v2 body, aggregate
// apps/checks are removed from state.body, and schema_version becomes 3.
// Callers must pass the SHA-256 from a recent PlanMySQLApplications report.
func ApplyMySQLApplications(parent context.Context, dsn, expectedSHA string) (ApplicationMigrationPlan, error) {
	var report ApplicationMigrationPlan
	expected, err := hex.DecodeString(strings.TrimSpace(expectedSHA))
	if err != nil || len(expected) != sha256.Size {
		return report, errors.New("expected source SHA-256 is required")
	}
	plan, err := PlanMySQLApplications(parent, dsn)
	if err != nil {
		return report, err
	}
	if !plan.RoundTripVerified {
		return report, errors.New("application migration plan failed")
	}
	if !bytes.Equal(expected, mustDecodeHex(plan.SourceSHA256)) {
		return report, errors.New("application migration source checksum mismatch")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL application apply configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 30*time.Second, 30*time.Second, 30*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL application apply connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = createMySQLApplicationSchema(ctx, db); err != nil {
		return report, errors.New("MySQL application apply schema unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return report, errors.New("MySQL application apply transaction unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL application apply source unavailable")
	}
	if version != 2 {
		return report, errors.New("application migration requires schema 2")
	}
	sum := sha256.Sum256(raw)
	if !bytes.Equal(sum[:], expected) {
		return report, errors.New("application migration source changed during apply")
	}
	if err = stageMySQLApplicationRoot(ctx, tx); err != nil {
		return report, errors.New("application root handoff failed")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET schema_version=3 WHERE id=1`); err != nil {
		return report, errors.New("application schema version switch failed")
	}
	if _, _, err = readMySQLApplicationV3(ctx, tx, true); err != nil {
		return report, errors.New("application migration reconciliation failed")
	}
	if err = tx.Commit(); err != nil {
		return report, errors.New("application migration commit not confirmed; inspect before retry")
	}
	plan.Status, plan.ReadOnly = "APPLIED", false
	return plan, nil
}

func mustDecodeHex(value string) []byte {
	b, err := hex.DecodeString(value)
	if err != nil {
		return nil
	}
	return b
}