package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errGrantUnavailable = errors.New("grant unavailable")

type grantDeltaKind int

const (
	grantDeltaNone grantDeltaKind = iota
	grantDeltaInsert
	grantDeltaRevoke
)

type grantDelta struct {
	kind  grantDeltaKind
	grant Grant
	audit []Event
}

func cloneStateSansGrantsAudit(st State) State {
	cloned := st
	cloned.Grants = nil
	cloned.Audit = nil
	return cloned
}

func detectGrantDelta(before, after State) (grantDelta, error) {
	if !reflectDeepEqualSansGrantAudit(before, after) {
		return grantDelta{}, errRelationalGrantDomain
	}
	if len(after.Grants) == len(before.Grants)+1 {
		seen := map[string]bool{}
		for _, g := range before.Grants {
			seen[g.ID] = true
		}
		var added Grant
		for _, g := range after.Grants {
			if !seen[g.ID] {
				if added.ID != "" {
					return grantDelta{}, errRelationalGrantDomain
				}
				added = g
			}
		}
		if added.ID == "" || !added.Active {
			return grantDelta{}, errRelationalGrantDomain
		}
		return grantDelta{kind: grantDeltaInsert, grant: added, audit: append([]Event(nil), after.Audit...)}, nil
	}
	if len(after.Grants) == len(before.Grants) {
		beforeByID := map[string]Grant{}
		for _, g := range before.Grants {
			beforeByID[g.ID] = g
		}
		revoked := 0
		changes := 0
		var target Grant
		for _, g := range after.Grants {
			prev, ok := beforeByID[g.ID]
			if !ok {
				return grantDelta{}, errRelationalGrantDomain
			}
			if equalGrantPayload(g, prev) {
				continue
			}
			if g.Active || !prev.Active || !equalGrantPayload(stripGrantActive(g), stripGrantActive(prev)) {
				return grantDelta{}, errRelationalGrantDomain
			}
			revoked++
			target = g
			changes++
		}
		if changes == 0 {
			return grantDelta{kind: grantDeltaNone}, nil
		}
		if revoked != 1 {
			return grantDelta{}, errRelationalGrantDomain
		}
		return grantDelta{kind: grantDeltaRevoke, grant: target, audit: append([]Event(nil), after.Audit...)}, nil
	}
	return grantDelta{}, errRelationalGrantDomain
}

func stripGrantActive(g Grant) Grant {
	g.Active = true
	return g
}

func equalGrantPayload(a, b Grant) bool {
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return equalApplicationJSON(ra, rb)
}

func reflectDeepEqualSansGrantAudit(before, after State) bool {
	return jsonEqualState(cloneStateSansGrantsAudit(before), cloneStateSansGrantsAudit(after))
}

func jsonEqualState(a, b State) bool {
	ra, err := json.Marshal(a)
	if err != nil {
		return false
	}
	rb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return equalApplicationJSON(ra, rb)
}

func findActiveGrantTuple(ctx context.Context, tx *sql.Tx, g Grant) (Grant, bool, error) {
	var payload []byte
	err := tx.QueryRowContext(ctx, `SELECT raw_payload FROM cognara_fg_grants WHERE tenant_id=? AND subject_type=? AND subject_id=? AND resource_type=? AND resource_id=? AND action=? AND scope=? AND active=1 FOR UPDATE`,
		g.TenantID, g.SubjectType, g.SubjectID, g.ResourceType, g.ResourceID, g.Action, g.Scope).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return Grant{}, false, nil
	}
	if err != nil {
		return Grant{}, false, err
	}
	var row Grant
	if json.Unmarshal(payload, &row) != nil {
		return Grant{}, false, errors.New("grant table payload invalid")
	}
	return row, true, nil
}

func insertMySQLGrantRow(ctx context.Context, tx *sql.Tx, g Grant) error {
	payload, err := json.Marshal(g)
	if err != nil {
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
	_, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_grants(tenant_id,grant_id,subject_type,subject_id,resource_type,resource_id,action,scope,active,metadata,raw_payload) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		g.TenantID, g.ID, g.SubjectType, g.SubjectID, g.ResourceType, g.ResourceID, g.Action, g.Scope, g.Active, string(meta), payload)
	return err
}

func updateMySQLGrantActive(ctx context.Context, tx *sql.Tx, g Grant) error {
	payload, err := json.Marshal(g)
	if err != nil {
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
	res, err := tx.ExecContext(ctx, `UPDATE cognara_fg_grants SET active=?,metadata=?,raw_payload=? WHERE tenant_id=? AND grant_id=?`,
		g.Active, string(meta), payload, g.TenantID, g.ID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil || affected != 1 {
		return errGrantUnavailable
	}
	return nil
}

func applyGrantDeltaV4(ctx context.Context, tx *sql.Tx, delta grantDelta) error {
	switch delta.kind {
	case grantDeltaNone:
		return nil
	case grantDeltaInsert:
		_, exists, err := findActiveGrantTuple(ctx, tx, delta.grant)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		if err = validateMySQLGrantPayload(delta.grant); err != nil {
			return err
		}
		if err = insertMySQLGrantRow(ctx, tx, delta.grant); err != nil {
			return err
		}
	case grantDeltaRevoke:
		var payload []byte
		err := tx.QueryRowContext(ctx, `SELECT raw_payload FROM cognara_fg_grants WHERE tenant_id=? AND grant_id=? FOR UPDATE`, delta.grant.TenantID, delta.grant.ID).Scan(&payload)
		if errors.Is(err, sql.ErrNoRows) {
			return errGrantUnavailable
		}
		if err != nil {
			return err
		}
		var row Grant
		if json.Unmarshal(payload, &row) != nil {
			return errors.New("grant table payload invalid")
		}
		if !row.Active {
			return nil
		}
		row.Active = false
		if err = updateMySQLGrantActive(ctx, tx, row); err != nil {
			return err
		}
	default:
		return errRelationalGrantDomain
	}
	if len(delta.audit) > 0 {
		var last uint64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
			return err
		}
		if err := insertMySQLAudit(ctx, tx, delta.audit, last); err != nil {
			return err
		}
	}
	return nil
}

func mutateMySQLGrantsV4(ctx context.Context, db *sql.DB, fn func(*State) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version); err != nil {
		return err
	}
	if version != 4 && version != 5 && version != 6 {
		return errors.New("relational grant writer requires schema 4, 5, or 6")
	}
	var before State
	switch version {
	case 4:
		before, _, err = readMySQLApplicationV4(ctx, tx, false)
	case 5:
		before, _, err = readMySQLApplicationV5(ctx, tx, false)
	case 6:
		before, _, err = readMySQLApplicationV6(ctx, tx, false)
	default:
		return errors.New("relational grant writer requires schema 4, 5, or 6")
	}
	if err != nil {
		return err
	}
	after := before
	after.Grants = append([]Grant(nil), before.Grants...)
	after.Audit = append([]Event(nil), before.Audit...)
	if err = fn(&after); err != nil {
		return err
	}
	delta, err := detectGrantDelta(before, after)
	if err != nil {
		return err
	}
	if delta.kind == grantDeltaNone && len(after.Audit) > 0 {
		return errRelationalGrantDomain
	}
	if err = applyGrantDeltaV4(ctx, tx, delta); err != nil {
		return err
	}
	switch version {
	case 4:
		_, _, err = readMySQLApplicationV4(ctx, tx, false)
	case 5:
		_, _, err = readMySQLApplicationV5(ctx, tx, false)
	case 6:
		_, _, err = readMySQLApplicationV6(ctx, tx, false)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}