package applications

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Preparatory root handoff inside a caller-owned rollback-only transaction.
// Remaining aggregate domains stay in state.body; the application envelope keeps
// only checks shape and an empty root sentinel. No schema3 activation is allowed.
// A caller must never commit this schema2 staging state or expose it to runtime.
func stageMySQLApplicationRoot(ctx context.Context, tx *sql.Tx) error {
	if err := copyMySQLApplicationSource(ctx, tx); err != nil {
		return err
	}
	var original []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&original); err != nil {
		return err
	}
	layout, err := splitApplicationLayout(original, time.Now().UTC())
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(layout.Root)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_application_envelope SET root_payload='{}' WHERE id=1`); err != nil {
		return err
	}
	got, err := loadStagedMySQLApplicationState(ctx, tx)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(original, got) {
		return errors.New("application root handoff reconciliation failed")
	}
	return nil
}

// Used only in isolated staging transactions, not by schema2 runtime readers.
func loadStagedMySQLApplicationState(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var version int
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1`).Scan(&version, &root); err != nil {
		return nil, err
	}
	if version != 2 {
		return nil, errors.New("application root staging requires schema 2")
	}
	return loadMySQLApplicationLayoutRoot(ctx, tx, time.Now().UTC(), root)
}

func rehearseMySQLApplicationRoot(ctx context.Context, db *sql.DB) (err error) {
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
	return stageMySQLApplicationRoot(ctx, tx)
}
