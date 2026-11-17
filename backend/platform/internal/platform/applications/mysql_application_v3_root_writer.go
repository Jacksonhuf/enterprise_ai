package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalApplicationDomain = errors.New("application and check changes require relational writers")

// Internal schema3 writer for the aggregate domains still owned by State.body
// (sessions, grants, settings, usage, receipts, exceptions). Application and
// check rows have dedicated writers; a callback changing them is rejected so the
// root never becomes a second authority. Audit keeps the schema2 rules: an
// operational callback sees no history and may only add events; a full callback
// may append but never alter existing events. Startup and HTTP stay gated.
func updateMySQLV3Root(ctx context.Context, db *sql.DB, operational bool, fn func(*State) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	var root []byte
	if err = tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &root); err != nil {
		return err
	}
	if version != 3 {
		return errors.New("relational root writer requires schema 3")
	}
	st, _, err := readMySQLApplicationV3(ctx, tx, !operational)
	if err != nil {
		return err
	}
	apps, err := json.Marshal(st.Apps)
	if err != nil {
		return err
	}
	checks, err := json.Marshal(st.Checks)
	if err != nil {
		return err
	}
	previous := append([]Event(nil), st.Audit...)
	if err = fn(&st); err != nil {
		return err
	}
	nextApps, err := json.Marshal(st.Apps)
	if err != nil {
		return err
	}
	nextChecks, err := json.Marshal(st.Checks)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(apps, nextApps) || !equalApplicationJSON(checks, nextChecks) {
		return errRelationalApplicationDomain
	}
	if operational {
		if len(st.Audit) > 0 {
			var last uint64
			if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
				return err
			}
			if err = insertMySQLAudit(ctx, tx, st.Audit, last); err != nil {
				return err
			}
		}
	} else if err = appendMySQLAudit(ctx, tx, previous, st.Audit); err != nil {
		return err
	}
	stored := st
	stored.Audit = nil
	body, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(body, &fields); err != nil {
		return err
	}
	delete(fields, "apps")
	delete(fields, "checks")
	if body, err = json.Marshal(fields); err != nil {
		return err
	}
	if body, err = preserveMySQLStateEnvelope(root, body); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(body)); err != nil {
		return err
	}
	if _, _, err = readMySQLApplicationV3(ctx, tx, false); err != nil {
		return err
	}
	return tx.Commit()
}
