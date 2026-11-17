package applications

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
)

type messageLayoutRow struct {
	TenantID  string          `json:"tenant_id"`
	SessionID string          `json:"session_id"`
	MessageID string          `json:"message_id"`
	Ordinal   uint            `json:"ordinal"`
	Payload   json.RawMessage `json:"payload_raw"`
}

type sessionLayoutRow struct {
	TenantID  string             `json:"tenant_id"`
	SessionID string             `json:"session_id"`
	AppID     string             `json:"app_id"`
	OwnerID   string             `json:"owner_id"`
	Payload   json.RawMessage    `json:"payload_raw"`
	Messages  []messageLayoutRow `json:"messages"`
}

type sessionLayout struct {
	Root           json.RawMessage `json:"root_payload"`
	Sessions       []sessionLayoutRow
	SessionsKind   string // absent, null, object
}

func splitSessionLayout(root []byte) (sessionLayout, error) {
	var layout sessionLayout
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return layout, errors.New("session layout root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants"} {
		if _, ok := object[key]; ok {
			return layout, errors.New("session layout requires schema4 root")
		}
	}
	layout.SessionsKind = "absent"
	value, ok := object["sessions"]
	if ok {
		layout.SessionsKind = "object"
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			layout.SessionsKind = "null"
		} else {
			var items map[string]json.RawMessage
			if json.Unmarshal(value, &items) != nil || items == nil {
				return layout, errors.New("session layout sessions must be an object")
			}
			keys := make([]string, 0, len(items))
			for key := range items {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				raw := items[key]
				var s Session
				if json.Unmarshal(raw, &s) != nil {
					return layout, errors.New("session layout item invalid")
				}
				if s.ID != key {
					return layout, errors.New("session layout key mismatch")
				}
				header := s
				messages := append([]Message(nil), s.Messages...)
				header.Messages = nil
				headerRaw, err := json.Marshal(header)
				if err != nil {
					return layout, err
				}
				row := sessionLayoutRow{TenantID: s.TenantID, SessionID: s.ID, AppID: s.AppID, OwnerID: s.OwnerID, Payload: headerRaw}
				for i, m := range messages {
					if m.SessionID != s.ID {
						return layout, errors.New("session layout message binding invalid")
					}
					payload, err := json.Marshal(m)
					if err != nil {
						return layout, err
					}
					row.Messages = append(row.Messages, messageLayoutRow{TenantID: s.TenantID, SessionID: s.ID, MessageID: m.ID, Ordinal: uint(i + 1), Payload: payload})
				}
				layout.Sessions = append(layout.Sessions, row)
			}
		}
	}
	delete(object, "sessions")
	var err error
	layout.Root, err = json.Marshal(object)
	return layout, err
}

func joinSessionLayout(layout sessionLayout) ([]byte, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(layout.Root, &root) != nil || root == nil {
		return nil, errors.New("invalid session root envelope")
	}
	if _, ok := root["sessions"]; ok {
		return nil, errors.New("duplicate session root authority")
	}
	sessions := map[string]json.RawMessage{}
	seenSession := map[string]bool{}
	seenMessage := map[string]bool{}
	for _, row := range layout.Sessions {
		if seenSession[row.SessionID] {
			return nil, errors.New("duplicate session layout row")
		}
		var header Session
		if json.Unmarshal(row.Payload, &header) != nil || header.ID != row.SessionID || header.TenantID != row.TenantID || header.AppID != row.AppID || header.OwnerID != row.OwnerID {
			return nil, errors.New("session projection does not match envelope")
		}
		seenSession[row.SessionID] = true
		messages := make([]Message, 0, len(row.Messages))
		for _, msg := range row.Messages {
			if msg.SessionID != row.SessionID || msg.TenantID != row.TenantID {
				return nil, errors.New("message row tenant/session mismatch")
			}
			if seenMessage[msg.MessageID] {
				return nil, errors.New("duplicate message id")
			}
			var m Message
			if json.Unmarshal(msg.Payload, &m) != nil || m.ID != msg.MessageID || m.SessionID != row.SessionID {
				return nil, errors.New("message projection invalid")
			}
			seenMessage[msg.MessageID] = true
			messages = append(messages, m)
		}
		header.Messages = messages
		raw, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}
		sessions[row.SessionID] = raw
	}
	switch layout.SessionsKind {
	case "absent":
		if len(sessions) != 0 {
			return nil, errors.New("non-object session container contains records")
		}
	case "null":
		return nil, errors.New("session layout null container unsupported")
	case "object":
		value, err := json.Marshal(sessions)
		if err != nil {
			return nil, err
		}
		root["sessions"] = value
	default:
		return nil, errors.New("unknown session container shape")
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if _, err = planSessionSource(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func verifySessionLayout(root []byte) error {
	layout, err := splitSessionLayout(root)
	if err != nil {
		return err
	}
	rebuilt, err := joinSessionLayout(layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(root, rebuilt) {
		return errors.New("session layout round-trip reconciliation failed")
	}
	return nil
}