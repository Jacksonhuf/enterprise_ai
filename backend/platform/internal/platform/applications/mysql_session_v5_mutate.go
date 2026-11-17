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
	if version != 5 {
		return errors.New("relational session writer requires schema 5")
	}
	before, _, err := readMySQLApplicationV5(ctx, tx, false)
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
	if _, _, err = readMySQLApplicationV5(ctx, tx, false); err != nil {
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