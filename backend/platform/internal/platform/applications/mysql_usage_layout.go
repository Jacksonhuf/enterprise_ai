package applications

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
)

type usageLayoutRow struct {
	TenantID   string          `json:"tenant_id"`
	EventID    string          `json:"event_id"`
	Ordinal    uint64          `json:"ordinal"`
	Unit       string          `json:"unit"`
	Capability string          `json:"capability"`
	OccurredAt string          `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload_raw"`
}

type usageLayout struct {
	Root      json.RawMessage `json:"root_payload"`
	Usage     []usageLayoutRow
	UsageKind string
}

func usageProjectionDefaults(item map[string]any) (unit, capability, occurred string) {
	unit, _ = item["unit"].(string)
	capability, _ = item["capability"].(string)
	occurred, _ = item["occurred_at"].(string)
	if unit == "" {
		unit = "-"
	}
	if capability == "" {
		capability = "-"
	}
	if occurred == "" {
		occurred = "-"
	}
	return unit, capability, occurred
}

func splitUsageLayout(root []byte) (usageLayout, error) {
	var layout usageLayout
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return layout, errors.New("usage layout root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions", "settings", "setting_revisions", "exceptions"} {
		if _, ok := object[key]; ok {
			return layout, errors.New("usage layout requires schema7 root")
		}
	}
	layout.UsageKind = "absent"
	if value, ok := object["usage"]; ok {
		layout.UsageKind = "array"
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			layout.UsageKind = "null"
		} else {
			var items []json.RawMessage
			if json.Unmarshal(value, &items) != nil {
				return layout, errors.New("usage layout usage must be an array")
			}
			for i, raw := range items {
				var item map[string]any
				decoder := json.NewDecoder(bytes.NewReader(raw))
				decoder.UseNumber()
				if decoder.Decode(&item) != nil || item == nil {
					return layout, errors.New("usage layout item invalid")
				}
				id, _ := item["event_id"].(string)
				tenant, _ := item["tenant_id"].(string)
				if !applicationPlanID(id, 512) || !applicationPlanID(tenant, 64) {
					return layout, errors.New("usage layout identity invalid")
				}
				unit, capability, occurred := usageProjectionDefaults(item)
				layout.Usage = append(layout.Usage, usageLayoutRow{
					TenantID: tenant, EventID: id, Ordinal: uint64(i) + 1,
					Unit: unit, Capability: capability, OccurredAt: occurred, Payload: append(json.RawMessage(nil), raw...),
				})
			}
		}
	}
	delete(object, "usage")
	var err error
	layout.Root, err = json.Marshal(object)
	return layout, err
}

func joinUsageLayout(layout usageLayout) ([]byte, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(layout.Root, &root) != nil || root == nil {
		return nil, errors.New("invalid usage root envelope")
	}
	if _, ok := root["usage"]; ok {
		return nil, errors.New("duplicate usage root authority")
	}
	rows := append([]usageLayoutRow(nil), layout.Usage...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Ordinal < rows[j].Ordinal })
	items := make([]json.RawMessage, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.EventID] {
			return nil, errors.New("duplicate usage layout row")
		}
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil {
			return nil, errors.New("usage projection does not match envelope")
		}
		id, _ := item["event_id"].(string)
		tenant, _ := item["tenant_id"].(string)
		if id != row.EventID || tenant != row.TenantID {
			return nil, errors.New("usage projection identity mismatch")
		}
		seen[row.EventID] = true
		items = append(items, row.Payload)
	}
	switch layout.UsageKind {
	case "absent", "null":
		if len(items) != 0 {
			return nil, errors.New("non-array usage container contains records")
		}
		if layout.UsageKind == "null" {
			root["usage"] = json.RawMessage(`null`)
		}
	case "array":
		value, err := json.Marshal(items)
		if err != nil {
			return nil, err
		}
		root["usage"] = value
	default:
		return nil, errors.New("unknown usage container shape")
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if _, err = planUsageSource(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func verifyUsageLayout(root []byte) error {
	layout, err := splitUsageLayout(root)
	if err != nil {
		return err
	}
	rebuilt, err := joinUsageLayout(layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(root, rebuilt) {
		return errors.New("usage layout round-trip reconciliation failed")
	}
	return nil
}