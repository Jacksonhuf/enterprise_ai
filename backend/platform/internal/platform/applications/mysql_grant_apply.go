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

// ApplyMySQLGrants migrates grants out of schema3 state.body into cognara_fg_grants
// and bumps schema_version to 4 atomically.
func ApplyMySQLGrants(parent context.Context, dsn, expectedSHA string) (GrantMigrationPlan, error) {
	var report GrantMigrationPlan
	expected, err := hex.DecodeString(strings.TrimSpace(expectedSHA))
	if err != nil || len(expected) != sha256.Size {
		return report, errors.New("expected source SHA-256 is required")
	}
	plan, err := PlanMySQLGrants(parent, dsn)
	if err != nil {
		return report, err
	}
	if !plan.RoundTripVerified {
		return report, errors.New("grant migration plan failed")
	}
	if !bytes.Equal(expected, mustDecodeHex(plan.SourceSHA256)) {
		return report, errors.New("grant migration source checksum mismatch")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" || cfg.MultiStatements {
		return report, errors.New("invalid MySQL grant apply configuration")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 30 * time.Second, 30*time.Second, 30*time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return report, errors.New("MySQL grant apply connection unavailable")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err = createMySQLGrantSchema(ctx, db); err != nil {
		return report, errors.New("MySQL grant apply schema unavailable")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return report, errors.New("MySQL grant apply transaction unavailable")
	}
	defer tx.Rollback()
	var version int
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &raw); err != nil {
		return report, errors.New("MySQL grant apply source unavailable")
	}
	if version != 3 {
		return report, errors.New("grant migration requires schema 3")
	}
	sum := sha256.Sum256(raw)
	if !bytes.Equal(sum[:], expected) {
		return report, errors.New("grant migration source changed during apply")
	}
	if err = stageMySQLGrantRoot(ctx, tx); err != nil {
		return report, errors.New("grant root handoff failed")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET schema_version=4 WHERE id=1`); err != nil {
		return report, errors.New("grant schema version switch failed")
	}
	if _, _, err = readMySQLApplicationV4(ctx, tx, true); err != nil {
		return report, errors.New("grant migration reconciliation failed")
	}
	if err = tx.Commit(); err != nil {
		return report, errors.New("grant migration commit not confirmed; inspect before retry")
	}
	plan.Status, plan.ReadOnly = "APPLIED", false
	return plan, nil
}