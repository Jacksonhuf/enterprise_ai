package applications

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Internal rehearsal against the actual source database, not a caller snapshot.
// Source writers serialize on the existing state row. Even success rolls back;
// runtime/schema3 activation uses ApplyMySQLApplications; rehearsal always rolls back.
func rehearseMySQLApplicationMigration(ctx context.Context, db *sql.DB) (err error) {
	var database string
	if err = db.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&database); err != nil {
		return err
	}
	if err = validateMySQLStateSchema(ctx, db, database); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if e := tx.Rollback(); err == nil && e != nil {
			err = e
		}
	}()
	return copyMySQLApplicationSource(ctx, tx)
}

// Caller must roll back this preparatory transaction. No version/root changes
// are made here. A future apply must integrate runtime and snapshot support first.
func copyMySQLApplicationSource(ctx context.Context, tx *sql.Tx) error {
	var version int
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &raw); err != nil {
		return err
	}
	if version != 2 {
		return errors.New("application migration requires schema 2")
	}
	if err := validateMySQLAuditSchema(ctx, tx); err != nil {
		return err
	}
	if _, err := verifyMySQLAuditStream(ctx, tx); err != nil {
		return err
	}
	return copyMySQLApplicationLayout(ctx, tx, raw, time.Now().UTC())
}
