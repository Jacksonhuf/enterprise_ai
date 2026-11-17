package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Internal rehearsal only: target tables must already exist and be empty.
// Always roll back, including success. No source writes, DDL, schema-version
// changes, or public apply path. Raw envelopes are the JSON authority; native
// JSON columns are checked projections, never used to rebuild opaque values.
func rehearseMySQLApplicationCopy(ctx context.Context, db *sql.DB, raw []byte, sampled time.Time) (err error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if e := tx.Rollback(); err == nil && e != nil {
			err = e
		}
	}()
	return copyMySQLApplicationLayout(ctx, tx, raw, sampled)
}

// Caller owns the transaction. Never commits, performs DDL or changes source state.
// Currently called only by the rollback rehearsal and isolated integration tests.
func copyMySQLApplicationLayout(ctx context.Context, tx *sql.Tx, raw []byte, sampled time.Time) error {
	layout, err := splitApplicationLayout(raw, sampled)
	if err != nil {
		return err
	}
	if err = applyMySQLApplicationLayout(ctx, tx, layout); err != nil {
		return err
	}
	rebuilt, err := loadMySQLApplicationLayout(ctx, tx, sampled)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(raw, rebuilt) {
		return errors.New("application database round-trip reconciliation failed")
	}
	return nil
}

// Caller must start from empty application tables. Writes raw envelopes only.
func applyMySQLApplicationLayout(ctx context.Context, tx *sql.Tx, layout applicationLayout) error {
	if err := validateMySQLApplicationSchema(ctx, tx); err != nil {
		return err
	}
	for table := range applicationSchemaSpecs() {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New("application layout target tables must be empty")
		}
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	var err error
	if err = exec(`INSERT INTO cognara_fg_application_envelope(id,root_payload,checks_kind) VALUES(1,?,?)`, []byte(layout.Root), layout.ChecksKind); err != nil {
		return err
	}
	for _, row := range layout.Apps {
		var a App
		if err = json.Unmarshal(row.Header, &a); err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_applications(tenant_id,app_id,agent_id,owner_id,current_release_id,updated_at,metadata,releases_is_null,raw_payload) VALUES(?,?,?,?,NULL,?,?,?,?)`, row.TenantID, row.AppID, a.AgentID, a.OwnerID, a.UpdatedAt, string(row.Header), row.ReleasesNull, []byte(row.Header)); err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_application_drafts(tenant_id,app_id,revision,config,raw_payload) VALUES(?,?,?,?,?)`, row.TenantID, row.AppID, a.Revision, string(row.Draft), []byte(row.Draft)); err != nil {
			return err
		}
		for ordinal, payload := range row.Releases {
			var r Release
			var fields map[string]json.RawMessage
			if err = json.Unmarshal(payload, &r); err != nil {
				return err
			}
			if err = json.Unmarshal(payload, &fields); err != nil {
				return err
			}
			if err = exec(`INSERT INTO cognara_fg_application_releases(tenant_id,app_id,release_id,ordinal,snapshot,published_at,published_by,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?)`, row.TenantID, row.AppID, r.ID, ordinal, string(fields["config_snapshot"]), r.PublishedAt, r.PublishedBy, string(payload), []byte(payload)); err != nil {
				return err
			}
		}
		if a.CurrentReleaseID != nil {
			if err = exec(`UPDATE cognara_fg_applications SET current_release_id=? WHERE tenant_id=? AND app_id=?`, *a.CurrentReleaseID, row.TenantID, row.AppID); err != nil {
				return err
			}
		}
	}
	for _, row := range layout.Checks {
		var c Check
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(row.Payload, &c); err != nil {
			return err
		}
		if err = json.Unmarshal(row.Payload, &fields); err != nil {
			return err
		}
		var expires string
		if err = json.Unmarshal(fields["expires_at"], &expires); err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_application_checks(tenant_id,check_id,app_id,draft_revision,user_id,expires_at,validation_id,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?)`, row.TenantID, row.CheckID, row.AppID, c.Revision, c.UserID, expires, c.ValidationID, string(row.Payload), []byte(row.Payload)); err != nil {
			return err
		}
	}
	return verifyMySQLApplicationProjections(ctx, tx, layout)
}

// Validate native SQL projections against the raw envelopes used to reconstruct
// state. JSON comparison here is only projection validation, not lossless proof.
func verifyMySQLApplicationProjections(ctx context.Context, q applicationSchemaQuery, layout applicationLayout) (err error) {
	expected := map[string]int{"cognara_fg_applications": len(layout.Apps), "cognara_fg_application_drafts": len(layout.Apps), "cognara_fg_application_releases": 0, "cognara_fg_application_checks": len(layout.Checks), "cognara_fg_application_envelope": 1}
	for _, a := range layout.Apps {
		expected["cognara_fg_application_releases"] += len(a.Releases)
	}
	for table, want := range expected {
		var n int
		if err = q.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
			return err
		}
		if n != want {
			return errors.New("application projection record count differs")
		}
	}
	// Reload the persisted payloads through the same transaction. MySQL JSON
	// normalization is compared by MySQL itself, without float64 decoding.
	verify := func(query string, args ...any) error {
		var n int
		if e := q.QueryRowContext(ctx, query, args...).Scan(&n); e != nil {
			return e
		}
		if n != 1 {
			return errors.New("application copy reconciliation failed")
		}
		return nil
	}
	for _, row := range layout.Apps {
		var a App
		if err = json.Unmarshal(row.Header, &a); err != nil {
			return err
		}
		if err = verify(`SELECT COUNT(*) FROM cognara_fg_applications WHERE tenant_id=? AND app_id=? AND agent_id=? AND owner_id=? AND current_release_id <=> ? AND updated_at=? AND metadata=CAST(? AS JSON) AND releases_is_null=?`, row.TenantID, row.AppID, a.AgentID, a.OwnerID, a.CurrentReleaseID, a.UpdatedAt, string(row.Header), row.ReleasesNull); err != nil {
			return err
		}
		if err = verify(`SELECT COUNT(*) FROM cognara_fg_application_drafts WHERE tenant_id=? AND app_id=? AND revision=? AND config=CAST(? AS JSON)`, row.TenantID, row.AppID, a.Revision, string(row.Draft)); err != nil {
			return err
		}
		for ordinal, payload := range row.Releases {
			var r Release
			var fields map[string]json.RawMessage
			json.Unmarshal(payload, &r)
			json.Unmarshal(payload, &fields)
			if err = verify(`SELECT COUNT(*) FROM cognara_fg_application_releases WHERE tenant_id=? AND app_id=? AND release_id=? AND ordinal=? AND snapshot=CAST(? AS JSON) AND published_at=? AND published_by=? AND metadata=CAST(? AS JSON)`, row.TenantID, row.AppID, r.ID, ordinal, string(fields["config_snapshot"]), r.PublishedAt, r.PublishedBy, string(payload)); err != nil {
				return err
			}
		}
	}
	for _, row := range layout.Checks {
		var c Check
		var fields map[string]json.RawMessage
		json.Unmarshal(row.Payload, &c)
		json.Unmarshal(row.Payload, &fields)
		var expires string
		json.Unmarshal(fields["expires_at"], &expires)
		if err = verify(`SELECT COUNT(*) FROM cognara_fg_application_checks WHERE tenant_id=? AND check_id=? AND app_id=? AND draft_revision=? AND user_id=? AND expires_at=? AND validation_id=? AND metadata=CAST(? AS JSON)`, row.TenantID, row.CheckID, row.AppID, c.Revision, c.UserID, expires, c.ValidationID, string(row.Payload)); err != nil {
			return err
		}
	}
	return nil
}
