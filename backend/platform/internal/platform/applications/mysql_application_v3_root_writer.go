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
// may append but never alter existing events. Store.Update* routes schema3 here;
// OpenMySQL accepts schema3 only after a validated handoff (apply, import, or tests).
func updateMySQLV3Root(ctx context.Context, db *sql.DB, operational bool, fn func(*State) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = updateMySQLV3RootTx(ctx, tx, operational, fn); err != nil {
		return err
	}
	return tx.Commit()
}

// The caller commits; the returned State is the committed view including any
// new audit events the callback added.
func updateMySQLV3RootTx(ctx context.Context, tx *sql.Tx, operational bool, fn func(*State) error) (State, error) {
	var zero State
	var version int
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version, &root); err != nil {
		return zero, err
	}
	if !mySQLRelationalRuntimeVersion(version) {
		return zero, errors.New("relational root writer requires schema 3, 4, 5, 6, 7, or 8")
	}
	var st State
	var err error
	switch version {
	case 3:
		st, _, err = readMySQLApplicationV3(ctx, tx, !operational)
	case 4:
		st, _, err = readMySQLApplicationV4(ctx, tx, !operational)
	case 5:
		st, _, err = readMySQLApplicationV5(ctx, tx, !operational)
	case 6:
		st, _, err = readMySQLApplicationV6(ctx, tx, !operational)
	case 7:
		st, _, err = readMySQLApplicationV7(ctx, tx, !operational)
	case 8:
		st, _, err = readMySQLApplicationV8(ctx, tx, !operational)
	default:
		return zero, errors.New("relational root writer schema unsupported")
	}
	if err != nil {
		return zero, err
	}
	beforeSessions := cloneSessionMap(st.Sessions)
	grantsBefore, err := json.Marshal(st.Grants)
	if err != nil {
		return zero, err
	}
	sessionsBefore, err := json.Marshal(st.Sessions)
	if err != nil {
		return zero, err
	}
	settingsBefore, err := json.Marshal(st.Settings)
	if err != nil {
		return zero, err
	}
	revisionsBefore, err := json.Marshal(st.SettingRevisions)
	if err != nil {
		return zero, err
	}
	exceptionsBefore, err := json.Marshal(st.Exceptions)
	if err != nil {
		return zero, err
	}
	usageBefore, err := json.Marshal(st.Usage)
	if err != nil {
		return zero, err
	}
	apps, err := json.Marshal(st.Apps)
	if err != nil {
		return zero, err
	}
	checks, err := json.Marshal(st.Checks)
	if err != nil {
		return zero, err
	}
	previous := append([]Event(nil), st.Audit...)
	unchanged := st
	unchanged.Audit = nil
	baseline, err := rootDomainsJSON(unchanged, version)
	if err != nil {
		return zero, err
	}
	if err = fn(&st); err != nil {
		return zero, err
	}
	nextApps, err := json.Marshal(st.Apps)
	if err != nil {
		return zero, err
	}
	nextChecks, err := json.Marshal(st.Checks)
	if err != nil {
		return zero, err
	}
	if !equalApplicationJSON(apps, nextApps) || !equalApplicationJSON(checks, nextChecks) {
		return zero, errRelationalApplicationDomain
	}
	grantsAfter, err := json.Marshal(st.Grants)
	if err != nil {
		return zero, err
	}
	if version == 4 && !equalApplicationJSON(grantsBefore, grantsAfter) {
		return zero, errRelationalGrantDomain
	}
	sessionsAfter, err := json.Marshal(st.Sessions)
	if err != nil {
		return zero, err
	}
	if version >= 5 && !equalApplicationJSON(sessionsBefore, sessionsAfter) {
		if err = applyMySQLSessionMapDelta(ctx, tx, beforeSessions, st.Sessions); err != nil {
			return zero, err
		}
	}
	if version == 5 && !equalApplicationJSON(grantsBefore, grantsAfter) {
		return zero, errRelationalGrantDomain
	}
	settingsAfter, err := json.Marshal(st.Settings)
	if err != nil {
		return zero, err
	}
	revisionsAfter, err := json.Marshal(st.SettingRevisions)
	if err != nil {
		return zero, err
	}
	if version == 6 && (!equalApplicationJSON(settingsBefore, settingsAfter) || !equalApplicationJSON(revisionsBefore, revisionsAfter)) {
		return zero, errRelationalSettingsDomain
	}
	exceptionsAfter, err := json.Marshal(st.Exceptions)
	if err != nil {
		return zero, err
	}
	if version == 7 && !equalApplicationJSON(exceptionsBefore, exceptionsAfter) {
		return zero, errRelationalExceptionDomain
	}
	usageAfter, err := json.Marshal(st.Usage)
	if err != nil {
		return zero, err
	}
	if version == 8 && !equalApplicationJSON(usageBefore, usageAfter) {
		var beforeItems []map[string]any
		if json.Unmarshal(usageBefore, &beforeItems) != nil {
			return zero, errRelationalUsageDomain
		}
		if err = applyMySQLUsageAppendDelta(ctx, tx, beforeItems, st.Usage); err != nil {
			return zero, err
		}
	}
	if operational {
		if len(st.Audit) > 0 {
			var last uint64
			if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
				return zero, err
			}
			if err = insertMySQLAudit(ctx, tx, st.Audit, last); err != nil {
				return zero, err
			}
		}
	} else if err = appendMySQLAudit(ctx, tx, previous, st.Audit); err != nil {
		return zero, err
	}
	stored := st
	stored.Audit = nil
	body, err := rootDomainsJSON(stored, version)
	if err != nil {
		return zero, err
	}
	if body, err = mergeRootJSON(root, baseline, body); err != nil {
		return zero, err
	}
	if body, err = preserveMySQLStateEnvelope(root, body); err != nil {
		return zero, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(body)); err != nil {
		return zero, err
	}
	switch version {
	case 3:
		_, _, err = readMySQLApplicationV3(ctx, tx, false)
	case 4:
		_, _, err = readMySQLApplicationV4(ctx, tx, false)
	case 5:
		_, _, err = readMySQLApplicationV5(ctx, tx, false)
	case 6:
		_, _, err = readMySQLApplicationV6(ctx, tx, false)
	case 7:
		_, _, err = readMySQLApplicationV7(ctx, tx, false)
	case 8:
		_, _, err = readMySQLApplicationV8(ctx, tx, false)
	}
	if err != nil {
		return zero, err
	}
	return st, nil
}

func rootDomainsJSON(st State, version int) ([]byte, error) {
	body, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	delete(fields, "apps")
	delete(fields, "checks")
	if version >= 4 {
		delete(fields, "grants")
	}
	if version >= 5 {
		delete(fields, "sessions")
	}
	if version >= 6 {
		delete(fields, "settings")
		delete(fields, "setting_revisions")
	}
	if version >= 7 {
		delete(fields, "exceptions")
	}
	if version >= 8 {
		delete(fields, "usage")
	}
	return json.Marshal(fields)
}

// Typed State decoding rounds large numbers and drops unknown keys. Untouched
// subtrees therefore keep the stored text, objects merge per key keeping keys
// the typed view never saw, and array elements reuse the stored element only
// when they equal a later old element in order, so removals never graft one
// element's unknown fields onto a neighbour; changed elements take the new value.
func mergeRootJSON(source, old, next []byte) ([]byte, error) {
	if len(source) > 0 && equalApplicationJSON(old, next) {
		return append([]byte(nil), source...), nil
	}
	var src, oldObj, nextObj map[string]json.RawMessage
	if json.Unmarshal(next, &nextObj) == nil && nextObj != nil && json.Unmarshal(source, &src) == nil && src != nil && json.Unmarshal(old, &oldObj) == nil && oldObj != nil {
		for key := range oldObj {
			if _, kept := nextObj[key]; !kept {
				delete(src, key)
			}
		}
		for key, value := range nextObj {
			original, stored := src[key]
			previous, known := oldObj[key]
			if !stored || !known {
				src[key] = value
				continue
			}
			merged, err := mergeRootJSON(original, previous, value)
			if err != nil {
				return nil, err
			}
			src[key] = merged
		}
		return json.Marshal(src)
	}
	var srcArr, oldArr, nextArr []json.RawMessage
	if json.Unmarshal(next, &nextArr) == nil && nextArr != nil && json.Unmarshal(source, &srcArr) == nil && json.Unmarshal(old, &oldArr) == nil && len(srcArr) == len(oldArr) {
		out := make([]json.RawMessage, len(nextArr))
		cursor := 0
		for i, value := range nextArr {
			out[i] = value
			for j := cursor; j < len(oldArr); j++ {
				if equalApplicationJSON(oldArr[j], value) {
					out[i] = srcArr[j]
					cursor = j + 1
					break
				}
			}
		}
		return json.Marshal(out)
	}
	return append([]byte(nil), next...), nil
}
