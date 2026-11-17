package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func applyMySQLSessionLayout(ctx context.Context, tx *sql.Tx, root []byte) error {
	if err := validateMySQLSessionSchema(ctx, tx); err != nil {
		return err
	}
	var sessions, messages int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM cognara_fg_sessions),(SELECT COUNT(*) FROM cognara_fg_session_messages)`).Scan(&sessions, &messages); err != nil {
		return err
	}
	if sessions != 0 || messages != 0 {
		return errors.New("session layout target tables must be empty")
	}
	layout, err := splitSessionLayout(root)
	if err != nil {
		return err
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range layout.Sessions {
		var header Session
		if err = json.Unmarshal(row.Payload, &header); err != nil {
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
		if err = exec(`INSERT INTO cognara_fg_sessions(tenant_id,session_id,app_id,owner_id,title_customized,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
			row.TenantID, row.SessionID, row.AppID, row.OwnerID, header.TitleCustomized, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
		for _, msg := range row.Messages {
			var m Message
			if err = json.Unmarshal(msg.Payload, &m); err != nil {
				return err
			}
			meta, err := json.Marshal(map[string]any{"role": m.Role, "created_at": m.CreatedAt, "is_completed": m.Completed})
			if err != nil {
				return err
			}
			if err = exec(`INSERT INTO cognara_fg_session_messages(tenant_id,session_id,message_id,ordinal,metadata,raw_payload) VALUES(?,?,?,?,?,?)`,
				msg.TenantID, msg.SessionID, msg.MessageID, msg.Ordinal, string(meta), []byte(msg.Payload)); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyMySQLSessionSnapshot(ctx context.Context, tx *sql.Tx, rows []sessionLayoutRow) error {
	if err := validateMySQLSessionSchema(ctx, tx); err != nil {
		return err
	}
	var sessions, messages int
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM cognara_fg_sessions),(SELECT COUNT(*) FROM cognara_fg_session_messages)`).Scan(&sessions, &messages); err != nil {
		return err
	}
	if sessions != 0 || messages != 0 {
		return errors.New("session import target must be empty")
	}
	exec := func(query string, args ...any) error { _, e := tx.ExecContext(ctx, query, args...); return e }
	for _, row := range rows {
		var header Session
		if err := json.Unmarshal(row.Payload, &header); err != nil {
			return err
		}
		if header.ID != row.SessionID || header.TenantID != row.TenantID || header.AppID != row.AppID || header.OwnerID != row.OwnerID {
			return errors.New("session snapshot row identity mismatch")
		}
		meta, err := json.Marshal(map[string]any{
			"title": header.Title, "revision": header.Revision,
			"created_at": header.CreatedAt, "updated_at": header.UpdatedAt,
			"title_customized": header.TitleCustomized,
		})
		if err != nil {
			return err
		}
		if err = exec(`INSERT INTO cognara_fg_sessions(tenant_id,session_id,app_id,owner_id,title_customized,metadata,raw_payload) VALUES(?,?,?,?,?,?,?)`,
			row.TenantID, row.SessionID, row.AppID, row.OwnerID, header.TitleCustomized, string(meta), []byte(row.Payload)); err != nil {
			return err
		}
		for _, msg := range row.Messages {
			var m Message
			if err = json.Unmarshal(msg.Payload, &m); err != nil {
				return err
			}
			if m.ID != msg.MessageID || m.SessionID != row.SessionID {
				return errors.New("session message snapshot row identity mismatch")
			}
			meta, err := json.Marshal(map[string]any{"role": m.Role, "created_at": m.CreatedAt, "is_completed": m.Completed})
			if err != nil {
				return err
			}
			if err = exec(`INSERT INTO cognara_fg_session_messages(tenant_id,session_id,message_id,ordinal,metadata,raw_payload) VALUES(?,?,?,?,?,?)`,
				msg.TenantID, msg.SessionID, msg.MessageID, msg.Ordinal, string(meta), []byte(msg.Payload)); err != nil {
				return err
			}
		}
	}
	return nil
}

func rehearseMySQLSessionMigration(ctx context.Context, db *sql.DB, root []byte) (err error) {
	if err = createMySQLSessionSchema(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() {
		if e := tx.Rollback(); err == nil && e != nil {
			err = e
		}
	}()
	if err = applyMySQLSessionLayout(ctx, tx, root); err != nil {
		return err
	}
	layout, err := splitSessionLayout(root)
	if err != nil {
		return err
	}
	var sessionCount, messageCount int
	if err = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM cognara_fg_sessions),(SELECT COUNT(*) FROM cognara_fg_session_messages)`).Scan(&sessionCount, &messageCount); err != nil {
		return err
	}
	if sessionCount != len(layout.Sessions) {
		return errors.New("session copy count mismatch")
	}
	wantMessages := 0
	for _, row := range layout.Sessions {
		wantMessages += len(row.Messages)
	}
	if messageCount != wantMessages {
		return errors.New("session message copy count mismatch")
	}
	return nil
}