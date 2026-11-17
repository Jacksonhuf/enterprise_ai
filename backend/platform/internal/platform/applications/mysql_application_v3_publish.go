package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var errRelationalReleaseConflict = errors.New("relational current release changed")

// Internal persistence only. Caller must authorize the trusted tenant/actor.
// The snapshot is the locked raw draft, so extensions and numeric precision are
// kept; earlier releases are never rewritten. Consumes the check atomically.
func publishMySQLV3Release(ctx context.Context, db *sql.DB, tenant, id, actor, client, request, check, expected string, current *string, note string) (Release, error) {
	var zero Release
	if tenant == "" || id == "" || !applicationPlanID(actor, 512) || check == "" || expected == "" {
		return zero, errors.New("invalid relational publish request")
	}
	if request == "" {
		request = ID("request")
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
	if !mySQLRelationalRuntimeVersion(version) {
		return zero, errors.New("relational publish requires schema 3 or 4")
	}
	st, _, err := readMySQLApplicationRelational(ctx, tx, false, version)
	if err != nil {
		return zero, err
	}
	a, ok := st.Apps[id]
	if !ok || a.TenantID != tenant {
		return zero, errRelationalAppUnavailable
	}
	var header, draft []byte
	if err = tx.QueryRowContext(ctx, `SELECT a.raw_payload,d.raw_payload FROM cognara_fg_applications a JOIN cognara_fg_application_drafts d ON d.tenant_id=a.tenant_id AND d.app_id=a.app_id WHERE a.tenant_id=? AND a.app_id=? FOR UPDATE`, tenant, id).Scan(&header, &draft); err != nil {
		return zero, err
	}
	var locked int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_application_checks WHERE tenant_id=? AND check_id=? FOR UPDATE`, tenant, check).Scan(&locked)
	if err != nil {
		return zero, err
	}
	c, ok := st.Checks[check]
	if locked != 1 || !ok || c.AppID != id || c.UserID != actor || c.Revision != a.Revision || expected != a.Revision || !time.Now().Before(c.ExpiresAt) {
		return zero, errRelationalCheckInvalid
	}
	if (a.CurrentReleaseID == nil) != (current == nil) || (a.CurrentReleaseID != nil && *a.CurrentReleaseID != *current) {
		return zero, errRelationalReleaseConflict
	}
	r := Release{ID: ID("release"), AppID: id, Note: note, PublishedAt: Now(), PublishedBy: actor}
	if err = json.Unmarshal(draft, &r.Config); err != nil {
		return zero, err
	}
	payload, err := json.Marshal(r)
	if err != nil {
		return zero, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(payload, &fields); err != nil {
		return zero, err
	}
	fields["config_snapshot"] = draft
	if payload, err = json.Marshal(fields); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_application_releases(tenant_id,app_id,release_id,ordinal,snapshot,published_at,published_by,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?)`, tenant, id, r.ID, len(a.Releases), string(draft), r.PublishedAt, actor, string(payload), payload); err != nil {
		return zero, err
	}
	var head map[string]json.RawMessage
	if err = json.Unmarshal(header, &head); err != nil {
		return zero, err
	}
	head["current_release_id"], _ = json.Marshal(r.ID)
	if header, err = json.Marshal(head); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_applications SET current_release_id=?,releases_is_null=0,metadata=?,raw_payload=? WHERE tenant_id=? AND app_id=?`, r.ID, string(header), header, tenant, id); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM cognara_fg_application_checks WHERE tenant_id=? AND check_id=?`, tenant, check); err != nil {
		return zero, err
	}
	var last uint64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
		return zero, err
	}
	if err = insertMySQLAudit(ctx, tx, []Event{{ID: ID("event"), TenantID: tenant, ActorID: actor, ClientID: client, OccurredAt: r.PublishedAt, Action: "application.publish", ResourceID: id, Outcome: "success", RequestID: request}}, last); err != nil {
		return zero, err
	}
	if _, _, err = readMySQLApplicationRelational(ctx, tx, false, version); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return r, nil
}
