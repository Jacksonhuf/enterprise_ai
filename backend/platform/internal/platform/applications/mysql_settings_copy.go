package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func applyMySQLSettingsLayout(ctx context.Context, tx *sql.Tx, root []byte, tenant string) error {
	if err := validateMySQLSettingsSchema(ctx, tx); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_settings`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("settings layout target tables must be empty")
	}
	layout, err := splitSettingsLayout(root)
	if err != nil {
		return err
	}
	layout.InferredTenant = tenant
	for i := range layout.Settings {
		layout.Settings[i].TenantID = tenant
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range layout.Settings {
		meta, err := json.Marshal(map[string]any{"revision": row.Revision, "owner_id": row.OwnerID, "section": row.Section})
		if err != nil {
			return err
		}
		raw, err := json.Marshal(map[string]any{"value": json.RawMessage(row.Payload), "revision": row.Revision})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_settings(tenant_id,setting_key,owner_id,section,revision,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
			row.TenantID, row.SettingKey, row.OwnerID, row.Section, row.Revision, string(meta), raw); err != nil {
			return err
		}
	}
	return nil
}

func rehearseMySQLSettingsMigration(ctx context.Context, db *sql.DB, root []byte, tenant string) (err error) {
	if err = createMySQLSettingsSchema(ctx, db); err != nil {
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
	if err = applyMySQLSettingsLayout(ctx, tx, root, tenant); err != nil {
		return err
	}
	layout, err := splitSettingsLayout(root)
	if err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_settings`).Scan(&count); err != nil {
		return err
	}
	if count != len(layout.Settings) {
		return errors.New("settings copy count mismatch")
	}
	return nil
}