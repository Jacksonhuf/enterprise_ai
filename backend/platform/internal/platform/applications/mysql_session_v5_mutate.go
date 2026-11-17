package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func mutateMySQLSessionsV5(ctx context.Context, db *sql.DB, fn func(*State) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&version); err != nil {
		return err
	}
	if version != 5 && version != 6 && version != 7 {
		return errors.New("relational session writer requires schema 5, 6, or 7")
	}
	before, _, err := readMySQLApplicationRelational(ctx, tx, false, version)
	if err != nil {
		return err
	}
	after := before
	after.Sessions = cloneSessionMap(before.Sessions)
	after.Audit = append([]Event(nil), before.Audit...)
	if err = fn(&after); err != nil {
		return err
	}
	beforeRaw, _ := json.Marshal(before.Sessions)
	afterRaw, _ := json.Marshal(after.Sessions)
	if equalApplicationJSON(beforeRaw, afterRaw) && len(after.Audit) == len(before.Audit) {
		return nil
	}
	baseline := cloneStateSansSessionsAudit(before)
	nextBaseline := cloneStateSansSessionsAudit(after)
	if !jsonEqualState(baseline, nextBaseline) {
		return errRelationalSessionDomain
	}
	if err = applyMySQLSessionMapDelta(ctx, tx, before.Sessions, after.Sessions); err != nil {
		return err
	}
	if len(after.Audit) > len(before.Audit) {
		var last uint64
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no),0) FROM cognara_fg_audit`).Scan(&last); err != nil {
			return err
		}
		if err = insertMySQLAudit(ctx, tx, after.Audit[len(before.Audit):], last); err != nil {
			return err
		}
	}
	if _, _, err = readMySQLApplicationRelational(ctx, tx, false, version); err != nil {
		return err
	}
	return tx.Commit()
}

func cloneStateSansSessionsAudit(st State) State {
	cloned := st
	cloned.Sessions = nil
	cloned.Audit = nil
	return cloned
}