package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var errRelationalCheckInvalid = errors.New("relational release check invalid")

// Internal persistence only. Caller must authorize the trusted tenant/actor and
// complete reference validation before recording its validation_id. Like the
// legacy path, recording a check writes no audit event.
func recordMySQLV3Check(ctx context.Context, db *sql.DB, tenant, id, actor, expected, validation string, expires time.Time) (Check, error) {
	var zero Check
	if tenant == "" || id == "" || !applicationPlanID(actor, 512) || !applicationPlanID(expected, 512) || len(validation) > 512 || !expires.After(time.Now()) {
		return zero, errors.New("invalid relational check request")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version); err != nil {
		return zero, err
	}
	if version != 3 {
		return zero, errors.New("relational check writer requires schema 3")
	}
	st, _, err := readMySQLApplicationV3(ctx, tx, false)
	if err != nil {
		return zero, err
	}
	a, ok := st.Apps[id]
	if !ok || a.TenantID != tenant {
		return zero, errRelationalAppUnavailable
	}
	var revision string
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM cognara_fg_application_drafts WHERE tenant_id=? AND app_id=? FOR UPDATE`, tenant, id).Scan(&revision); err != nil {
		return zero, err
	}
	if revision != expected || a.Revision != expected {
		return zero, errRelationalDraftConflict
	}
	c := Check{ID: ID("check"), AppID: id, Revision: expected, UserID: actor, ExpiresAt: expires.UTC(), ValidationID: validation}
	payload, err := json.Marshal(c)
	if err != nil {
		return zero, err
	}
	var fields map[string]json.RawMessage
	var until string
	if err = json.Unmarshal(payload, &fields); err != nil {
		return zero, err
	}
	if err = json.Unmarshal(fields["expires_at"], &until); err != nil {
		return zero, err
	}
	// The legacy writer materializes an absent or null container as an object.
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_application_envelope SET checks_kind='object' WHERE id=1 AND checks_kind<>'object'`); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_application_checks(tenant_id,check_id,app_id,draft_revision,user_id,expires_at,validation_id,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?)`, tenant, c.ID, id, c.Revision, actor, until, validation, string(payload), payload); err != nil {
		return zero, err
	}
	if _, _, err = readMySQLApplicationV3(ctx, tx, false); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return c, nil
}
