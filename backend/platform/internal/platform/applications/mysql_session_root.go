package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var errRelationalSessionDomain = errors.New("session changes require relational writers")

func stageMySQLSessionRoot(ctx context.Context, tx *sql.Tx) error {
	var original []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1 FOR UPDATE`).Scan(&original); err != nil {
		return err
	}
	layout, err := splitSessionLayout(original)
	if err != nil {
		return err
	}
	if err := applyMySQLSessionLayout(ctx, tx, original); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE cognara_fg_state SET body=? WHERE id=1`, string(layout.Root)); err != nil {
		return err
	}
	got, err := loadStagedMySQLSessionState(ctx, tx, layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(original, got) {
		return errors.New("session root handoff reconciliation failed")
	}
	return nil
}

func scanSessionLayout(ctx context.Context, q applicationSchemaQuery) ([]sessionLayoutRow, error) {
	rows, err := q.QueryContext(ctx, `SELECT tenant_id,session_id,app_id,owner_id,raw_payload FROM cognara_fg_sessions ORDER BY tenant_id,session_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layout []sessionLayoutRow
	for rows.Next() {
		var tenant, sessionID, appID, ownerID, payload []byte
		if err = rows.Scan(&tenant, &sessionID, &appID, &ownerID, &payload); err != nil {
			return nil, err
		}
		layout = append(layout, sessionLayoutRow{
			TenantID: string(tenant), SessionID: string(sessionID), AppID: string(appID), OwnerID: string(ownerID),
			Payload: append(json.RawMessage(nil), payload...),
		})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for i := range layout {
		row := &layout[i]
		msgRows, err := q.QueryContext(ctx, `SELECT tenant_id,session_id,message_id,ordinal,raw_payload FROM cognara_fg_session_messages WHERE tenant_id=? AND session_id=? ORDER BY ordinal`,
			[]byte(row.TenantID), []byte(row.SessionID))
		if err != nil {
			return nil, err
		}
		for msgRows.Next() {
			var mTenant, mSession, messageID, raw []byte
			var ordinal uint
			if err = msgRows.Scan(&mTenant, &mSession, &messageID, &ordinal, &raw); err != nil {
				msgRows.Close()
				return nil, err
			}
			row.Messages = append(row.Messages, messageLayoutRow{
				TenantID: string(mTenant), SessionID: string(mSession), MessageID: string(messageID),
				Ordinal: ordinal, Payload: append(json.RawMessage(nil), raw...),
			})
		}
		msgRows.Close()
		if err = msgRows.Err(); err != nil {
			return nil, err
		}
	}
	return layout, nil
}

func loadStagedMySQLSessionState(ctx context.Context, tx *sql.Tx, layout sessionLayout) ([]byte, error) {
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version); err != nil {
		return nil, err
	}
	if version != 4 {
		return nil, errors.New("session root staging requires schema 4")
	}
	rows, err := scanSessionLayout(ctx, tx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(layout.Sessions) {
		return nil, errors.New("session staging row count mismatch")
	}
	byID := map[string]sessionLayoutRow{}
	for _, row := range rows {
		if _, exists := byID[row.SessionID]; exists {
			return nil, errors.New("duplicate staged session id")
		}
		byID[row.SessionID] = row
	}
	for _, expected := range layout.Sessions {
		row, ok := byID[expected.SessionID]
		if !ok || row.TenantID != expected.TenantID || !equalApplicationJSON(expected.Payload, row.Payload) || len(row.Messages) != len(expected.Messages) {
			return nil, errors.New("staged session row mismatch")
		}
		for i, msg := range expected.Messages {
			got := row.Messages[i]
			if got.MessageID != msg.MessageID || got.Ordinal != msg.Ordinal || !equalApplicationJSON(msg.Payload, got.Payload) {
				return nil, errors.New("staged session message mismatch")
			}
		}
	}
	layout.SessionsKind = "object"
	if len(rows) == 0 {
		layout.SessionsKind = "absent"
	}
	layout.Root, err = currentSessionStagingRoot(ctx, tx)
	if err != nil {
		return nil, err
	}
	return joinSessionLayout(layout)
}

func currentSessionStagingRoot(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	var root []byte
	if err := tx.QueryRowContext(ctx, `SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&root); err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return nil, errors.New("invalid session staging root")
	}
	if _, ok := object["sessions"]; ok {
		return nil, errors.New("session staging root still contains sessions")
	}
	return append([]byte(nil), root...), nil
}

func loadMySQLSessionsFromTable(ctx context.Context, q applicationSchemaQuery) (map[string]Session, error) {
	rows, err := scanSessionLayout(ctx, q)
	if err != nil {
		return nil, err
	}
	sessions := map[string]Session{}
	for _, row := range rows {
		var header Session
		if json.Unmarshal(row.Payload, &header) != nil || header.ID != row.SessionID || header.TenantID != row.TenantID {
			return nil, errors.New("session table payload invalid")
		}
		messages := make([]Message, 0, len(row.Messages))
		for _, msg := range row.Messages {
			var m Message
			if json.Unmarshal(msg.Payload, &m) != nil || m.ID != msg.MessageID || m.SessionID != row.SessionID {
				return nil, errors.New("session message table payload invalid")
			}
			messages = append(messages, m)
		}
		header.Messages = messages
		sessions[header.ID] = header
	}
	return sessions, nil
}