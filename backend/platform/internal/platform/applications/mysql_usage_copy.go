package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func applyMySQLUsageLayout(ctx context.Context, tx *sql.Tx, root []byte) error {
	if err := validateMySQLUsageSchema(ctx, tx); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_usage`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("usage layout target tables must be empty")
	}
	layout, err := splitUsageLayout(root)
	if err != nil {
		return err
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range layout.Usage {
		meta, err := json.Marshal(map[string]any{
			"unit": row.Unit, "capability": row.Capability, "occurred_at": row.OccurredAt, "ordinal": row.Ordinal,
		})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_usage(tenant_id,event_id,ordinal,unit,capability,occurred_at,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?)`,
			row.TenantID, row.EventID, row.Ordinal, row.Unit, row.Capability, row.OccurredAt, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
	}
	return nil
}

func rehearseMySQLUsageMigration(ctx context.Context, db *sql.DB, root []byte) (err error) {
	if err = createMySQLUsageSchema(ctx, db); err != nil {
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
	if err = applyMySQLUsageLayout(ctx, tx, root); err != nil {
		return err
	}
	layout, err := splitUsageLayout(root)
	if err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_usage`).Scan(&count); err != nil {
		return err
	}
	if count != len(layout.Usage) {
		return errors.New("usage copy count mismatch")
	}
	return nil
}