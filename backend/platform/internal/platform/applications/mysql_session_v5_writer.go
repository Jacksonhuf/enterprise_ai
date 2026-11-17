package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errSessionUnavailable = errors.New("session unavailable")

func cloneSessionMap(in map[string]Session) map[string]Session {
	if len(in) == 0 {
		return map[string]Session{}
	}
	out := make(map[string]Session, len(in))
	for id, s := range in {
		copy := s
		copy.Messages = append([]Message(nil), s.Messages...)
		out[id] = copy
	}
	return out
}

func insertMySQLSessionRow(ctx context.Context, tx *sql.Tx, s Session) error {
	header := s
	header.Messages = nil
	payload, err := json.Marshal(header)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{
		"title": header.Title, "revision": header.Revision,
		"created_at": header.CreatedAt, "updated_at": header.UpdatedAt,
		"title_customized": header.TitleCustomized,
	})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_sessions(tenant_id,session_id,app_id,owner_id,title_customized,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
		s.TenantID, s.ID, s.AppID, s.OwnerID, s.TitleCustomized, string(meta), payload)
	return err
}

func insertMySQLSessionMessageRow(ctx context.Context, tx *sql.Tx, tenantID, sessionID string, ordinal uint, m Message) error {
	payload, err := json.Marshal(m)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{"role": m.Role, "created_at": m.CreatedAt, "is_completed": m.Completed})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cognara_fg_session_messages(tenant_id,session_id,message_id,ordinal,metadata,raw_payload) VALUES(?,?,?,?,?,?)`,
		tenantID, sessionID, m.ID, ordinal, string(meta), payload)
	return err
}

func deleteMySQLSession(ctx context.Context, tx *sql.Tx, tenantID, sessionID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM cognara_fg_session_messages WHERE tenant_id=? AND session_id=?`, tenantID, sessionID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM cognara_fg_sessions WHERE tenant_id=? AND session_id=?`, tenantID, sessionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil || affected != 1 {
		return errSessionUnavailable
	}
	return nil
}

func writeMySQLSession(ctx context.Context, tx *sql.Tx, s Session) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM cognara_fg_sessions WHERE tenant_id=? AND session_id=?`, s.TenantID, s.ID).Scan(&count); err != nil {
		return err
	}
	header := s
	header.Messages = nil
	payload, err := json.Marshal(header)
	if err != nil {
		return err
	}
	meta, err := json.Marshal(map[string]any{
		"title": header.Title, "revision": header.Revision,
		"created_at": header.CreatedAt, "updated_at": header.UpdatedAt,
		"title_customized": header.TitleCustomized,
	})
	if err != nil {
		return err
	}
	if count == 0 {
		if err = insertMySQLSessionRow(ctx, tx, s); err != nil {
			return err
		}
	} else {
		res, err := tx.ExecContext(ctx, `UPDATE cognara_fg_sessions SET app_id=?,owner_id=?,title_customized=?,metadata=?,raw_payload=? WHERE tenant_id=? AND session_id=?`,
			s.AppID, s.OwnerID, s.TitleCustomized, string(meta), payload, s.TenantID, s.ID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return errSessionUnavailable
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM cognara_fg_session_messages WHERE tenant_id=? AND session_id=?`, s.TenantID, s.ID); err != nil {
		return err
	}
	for i, m := range s.Messages {
		if m.SessionID != s.ID {
			return errors.New("session message binding invalid")
		}
		if err = insertMySQLSessionMessageRow(ctx, tx, s.TenantID, s.ID, uint(i+1), m); err != nil {
			return err
		}
	}
	return nil
}

func applyMySQLSessionMapDelta(ctx context.Context, tx *sql.Tx, before, after map[string]Session) error {
	if before == nil {
		before = map[string]Session{}
	}
	if after == nil {
		after = map[string]Session{}
	}
	for id, prev := range before {
		if _, ok := after[id]; !ok {
			if err := deleteMySQLSession(ctx, tx, prev.TenantID, id); err != nil {
				return err
			}
		}
	}
	for id, next := range after {
		prev, ok := before[id]
		if !ok {
			if err := writeMySQLSession(ctx, tx, next); err != nil {
				return err
			}
			continue
		}
		prevRaw, _ := json.Marshal(prev)
		nextRaw, _ := json.Marshal(next)
		if !equalApplicationJSON(prevRaw, nextRaw) {
			if err := writeMySQLSession(ctx, tx, next); err != nil {
				return err
			}
		}
	}
	return nil
}