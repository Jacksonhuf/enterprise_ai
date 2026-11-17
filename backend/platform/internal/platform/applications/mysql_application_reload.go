package applications

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Reconstruct only from stored raw envelopes, never from MySQL's normalized JSON.
// Caller supplies a consistent transaction; no cached layout/source is consulted.
func loadMySQLApplicationLayout(ctx context.Context, tx *sql.Tx, sampled time.Time) ([]byte, error) {
	return loadMySQLApplicationLayoutRoot(ctx, tx, sampled, nil)
}

func scanMySQLApplicationLayout(ctx context.Context, tx *sql.Tx) (applicationLayout, error) {
	var layout applicationLayout
	if err := validateMySQLApplicationSchema(ctx, tx); err != nil {
		return layout, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT root_payload,checks_kind FROM cognara_fg_application_envelope WHERE id=1`).Scan(&layout.Root, &layout.ChecksKind); err != nil {
		return layout, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT a.tenant_id,a.app_id,a.raw_payload,d.raw_payload,a.releases_is_null FROM cognara_fg_applications a JOIN cognara_fg_application_drafts d ON d.tenant_id=a.tenant_id AND d.app_id=a.app_id ORDER BY a.tenant_id,a.app_id`)
	if err != nil {
		return layout, err
	}
	parents := map[string]int{}
	for rows.Next() {
		var a applicationLayoutRow
		if err = rows.Scan(&a.TenantID, &a.AppID, &a.Header, &a.Draft, &a.ReleasesNull); err != nil {
			rows.Close()
			return layout, err
		}
		parents[a.AppID] = len(layout.Apps)
		layout.Apps = append(layout.Apps, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return layout, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT tenant_id,app_id,ordinal,raw_payload FROM cognara_fg_application_releases ORDER BY tenant_id,app_id,ordinal`)
	if err != nil {
		return layout, err
	}
	for rows.Next() {
		var tenant, id string
		var ordinal uint64
		var payload []byte
		if err = rows.Scan(&tenant, &id, &ordinal, &payload); err != nil {
			rows.Close()
			return layout, err
		}
		index, ok := parents[id]
		if !ok || layout.Apps[index].TenantID != tenant || ordinal != uint64(len(layout.Apps[index].Releases)) {
			rows.Close()
			return layout, errors.New("application release parent or ordinal differs")
		}
		layout.Apps[index].Releases = append(layout.Apps[index].Releases, payload)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return layout, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT tenant_id,app_id,check_id,raw_payload FROM cognara_fg_application_checks ORDER BY tenant_id,check_id`)
	if err != nil {
		return layout, err
	}
	for rows.Next() {
		var c applicationCheckLayoutRow
		if err = rows.Scan(&c.TenantID, &c.AppID, &c.CheckID, &c.Payload); err != nil {
			rows.Close()
			return layout, err
		}
		layout.Checks = append(layout.Checks, c)
	}
	err = rows.Err()
	rows.Close()
	return layout, err
}

func loadMySQLApplicationLayoutRoot(ctx context.Context, tx *sql.Tx, sampled time.Time, root []byte) ([]byte, error) {
	layout, err := scanMySQLApplicationLayout(ctx, tx)
	if err != nil {
		return nil, err
	}
	if root != nil {
		if !equalApplicationJSON(layout.Root, []byte(`{}`)) {
			return nil, errors.New("duplicate application root authority")
		}
		layout.Root = append([]byte(nil), root...)
	}
	rebuilt, err := joinApplicationLayout(layout, sampled)
	if err != nil {
		return nil, err
	}
	if err = verifyMySQLApplicationProjections(ctx, tx, layout); err != nil {
		return nil, err
	}
	return rebuilt, nil
}
