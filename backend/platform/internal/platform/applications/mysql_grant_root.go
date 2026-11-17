package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalGrantDomain = errors.New("grant changes require relational writers")

func stageMySQLGrantRoot(ctx context.Context, tx *sql.Tx) error {
	var original []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&original); err != nil {
		return err
	}
	layout, err := splitGrantLayout(original)
	if err != nil {
		return err
	}
	if err := applyMySQLGrantLayout(ctx, tx, original); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(layout.Root)); err != nil {
		return err
	}
	got, err := loadStagedMySQLGrantState(ctx, tx, layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(original, got) {
		return errors.New("grant root handoff reconciliation failed")
	}
	return nil
}

func loadStagedMySQLGrantState(ctx context.Context, tx *sql.Tx, layout grantLayout) ([]byte, error) {
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return nil, err
	}
	if version != 3 {
		return nil, errors.New("grant root staging requires schema 3")
	}
	rows, err := scanGrantLayout(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(layout.Grants) {
		return nil, errors.New("grant staging row count mismatch")
	}
	byID := map[string]grantLayoutRow{}
	for _, row := range rows {
		if _, exists := byID[row.GrantID]; exists {
			return nil, errors.New("duplicate staged grant id")
		}
		byID[row.GrantID] = row
	}
	for _, expected := range layout.Grants {
		row, ok := byID[expected.GrantID]
		if !ok || row.TenantID != expected.TenantID || !equalApplicationJSON(expected.Payload, row.Payload) {
			return nil, errors.New("staged grant row mismatch")
		}
	}
	layout.Root, err = currentGrantStagingRoot(ctx, tx)
	if err != nil {
		return nil, err
	}
	return joinGrantLayout(layout)
}

func currentGrantStagingRoot(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return nil, errors.New("invalid grant staging root")
	}
	if _, ok := object["grants"]; ok {
		return nil, errors.New("grant staging root still contains grants")
	}
	return append([]byte(nil), root...), nil
}

func loadMySQLGrantsFromTable(ctx context.Context, q applicationSchemaQuery) ([]Grant, error) {
	rows, err := scanGrantLayout(ctx, q)
	if err != nil {
		return nil, err
	}
	grants := make([]Grant, 0, len(rows))
	for _, row := range rows {
		var g Grant
		if json.Unmarshal(row.Payload, &g) != nil || g.ID != row.GrantID || g.TenantID != row.TenantID {
			return nil, errors.New("grant table payload invalid")
		}
		grants = append(grants, g)
	}
	return grants, nil
}