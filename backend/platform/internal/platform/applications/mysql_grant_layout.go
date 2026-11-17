package applications

import (
	"bytes"
	"encoding/json"
	"errors"
)

type grantLayoutRow struct {
	TenantID string          `json:"tenant_id"`
	GrantID  string          `json:"grant_id"`
	Payload  json.RawMessage `json:"payload_raw"`
}

type grantLayout struct {
	Root       json.RawMessage `json:"root_payload"`
	Grants     []grantLayoutRow
	GrantsKind string // absent, null, array
}

func splitGrantLayout(root []byte) (grantLayout, error) {
	var layout grantLayout
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return layout, errors.New("grant layout root invalid")
	}
	if _, ok := object["apps"]; ok {
		return layout, errors.New("grant layout requires schema3 root")
	}
	if _, ok := object["checks"]; ok {
		return layout, errors.New("grant layout requires schema3 root")
	}
	layout.GrantsKind = "absent"
	if value, ok := object["grants"]; ok {
		layout.GrantsKind = "array"
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			layout.GrantsKind = "null"
		} else {
			var items []json.RawMessage
			if json.Unmarshal(value, &items) != nil {
				return layout, errors.New("grant layout grants must be an array")
			}
			for _, raw := range items {
				var g Grant
				if json.Unmarshal(raw, &g) != nil {
					return layout, errors.New("grant layout item invalid")
				}
				layout.Grants = append(layout.Grants, grantLayoutRow{TenantID: g.TenantID, GrantID: g.ID, Payload: append(json.RawMessage(nil), raw...)})
			}
		}
	}
	delete(object, "grants")
	var err error
	layout.Root, err = json.Marshal(object)
	return layout, err
}

func joinGrantLayout(layout grantLayout) ([]byte, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(layout.Root, &root) != nil || root == nil {
		return nil, errors.New("invalid grant root envelope")
	}
	if _, ok := root["grants"]; ok {
		return nil, errors.New("duplicate grant root authority")
	}
	items := make([]json.RawMessage, 0, len(layout.Grants))
	seen := map[string]bool{}
	for _, row := range layout.Grants {
		if seen[row.GrantID] {
			return nil, errors.New("duplicate grant layout row")
		}
		var g Grant
		if json.Unmarshal(row.Payload, &g) != nil || g.TenantID != row.TenantID || g.ID != row.GrantID {
			return nil, errors.New("grant projection does not match envelope")
		}
		seen[row.GrantID] = true
		items = append(items, row.Payload)
	}
	switch layout.GrantsKind {
	case "absent", "null":
		if len(items) != 0 {
			return nil, errors.New("non-array grant container contains records")
		}
		if layout.GrantsKind == "null" {
			root["grants"] = json.RawMessage(`null`)
		}
	case "array":
		value, err := json.Marshal(items)
		if err != nil {
			return nil, err
		}
		root["grants"] = value
	default:
		return nil, errors.New("unknown grant container shape")
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if _, err = planGrantSource(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func verifyGrantLayout(root []byte) error {
	layout, err := splitGrantLayout(root)
	if err != nil {
		return err
	}
	rebuilt, err := joinGrantLayout(layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(root, rebuilt) {
		return errors.New("grant layout round-trip reconciliation failed")
	}
	return nil
}