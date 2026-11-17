package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func applyMySQLGrantLayout(ctx context.Context, tx *sql.Tx, root []byte) error {
	if err := validateMySQLGrantSchema(ctx, tx); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_grants`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("grant layout target table must be empty")
	}
	layout, err := splitGrantLayout(root)
	if err != nil {
		return err
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range layout.Grants {
		var g Grant
		if err = json.Unmarshal(row.Payload, &g); err != nil {
			return err
		}
		meta, err := json.Marshal(map[string]any{
			"subject_type": g.SubjectType, "subject_id": g.SubjectID,
			"resource_type": g.ResourceType, "resource_id": g.ResourceID,
			"action": g.Action, "scope": g.Scope, "active": g.Active,
		})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_grants(tenant_id,grant_id,subject_type,subject_id,resource_type,resource_id,action,scope,active,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			g.TenantID, g.ID, g.SubjectType, g.SubjectID, g.ResourceType, g.ResourceID, g.Action, g.Scope, g.Active, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
	}
	return nil
}

func applyMySQLGrantSnapshot(ctx context.Context, tx *sql.Tx, rows []grantLayoutRow) error {
	if err := validateMySQLGrantSchema(ctx, tx); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_grants`).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("grant import target must be empty")
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range rows {
		var g Grant
		if err := json.Unmarshal(row.Payload, &g); err != nil {
			return err
		}
		if g.ID != row.GrantID || g.TenantID != row.TenantID {
			return errors.New("grant snapshot row identity mismatch")
		}
		meta, err := json.Marshal(map[string]any{
			"subject_type": g.SubjectType, "subject_id": g.SubjectID,
			"resource_type": g.ResourceType, "resource_id": g.ResourceID,
			"action": g.Action, "scope": g.Scope, "active": g.Active,
		})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_grants(tenant_id,grant_id,subject_type,subject_id,resource_type,resource_id,action,scope,active,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			g.TenantID, g.ID, g.SubjectType, g.SubjectID, g.ResourceType, g.ResourceID, g.Action, g.Scope, g.Active, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
	}
	return nil
}

func rehearseMySQLGrantMigration(ctx context.Context, db *sql.DB, root []byte) (err error) {
	if err = createMySQLGrantSchema(ctx, db); err != nil {
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
	if err = applyMySQLGrantLayout(ctx, tx, root); err != nil {
		return err
	}
	layout, err := splitGrantLayout(root)
	if err != nil {
		return err
	}
	rows, err := scanGrantLayout(ctx, tx)
	if err != nil {
		return err
	}
	if len(rows) != len(layout.Grants) {
		return errors.New("grant copy count mismatch")
	}
	return nil
}