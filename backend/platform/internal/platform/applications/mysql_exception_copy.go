package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func applyMySQLExceptionLayout(ctx context.Context, tx *sql.Tx, root []byte) error {
	if err := validateMySQLExceptionSchema(ctx, tx); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_exceptions`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("exception layout target tables must be empty")
	}
	layout, err := splitExceptionLayout(root)
	if err != nil {
		return err
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range layout.Exceptions {
		meta, err := json.Marshal(map[string]any{"status": row.Status, "request_id": row.RequestID, "ordinal": row.Ordinal})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_exceptions(tenant_id,exception_id,ordinal,status,request_id,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
			row.TenantID, row.ExceptionID, row.Ordinal, row.Status, row.RequestID, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
	}
	return nil
}

func rehearseMySQLExceptionMigration(ctx context.Context, db *sql.DB, root []byte) (err error) {
	if err = createMySQLExceptionSchema(ctx, db); err != nil {
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
	if err = applyMySQLExceptionLayout(ctx, tx, root); err != nil {
		return err
	}
	layout, err := splitExceptionLayout(root)
	if err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_exceptions`).Scan(&count); err != nil {
		return err
	}
	if count != len(layout.Exceptions) {
		return errors.New("exception copy count mismatch")
	}
	return nil
}