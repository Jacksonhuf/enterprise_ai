package applications

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
)

type exceptionLayoutRow struct {
	TenantID    string          `json:"tenant_id"`
	ExceptionID string          `json:"exception_id"`
	Ordinal     uint64          `json:"ordinal"`
	Status      string          `json:"status"`
	RequestID   string          `json:"request_id"`
	Payload     json.RawMessage `json:"payload_raw"`
}

type exceptionLayout struct {
	Root            json.RawMessage `json:"root_payload"`
	Exceptions      []exceptionLayoutRow
	ExceptionsKind  string
}

func splitExceptionLayout(root []byte) (exceptionLayout, error) {
	var layout exceptionLayout
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return layout, errors.New("exception layout root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions", "settings", "setting_revisions"} {
		if _, ok := object[key]; ok {
			return layout, errors.New("exception layout requires schema6 root")
		}
	}
	layout.ExceptionsKind = "absent"
	if value, ok := object["exceptions"]; ok {
		layout.ExceptionsKind = "array"
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			layout.ExceptionsKind = "null"
		} else {
			var items []json.RawMessage
			if json.Unmarshal(value, &items) != nil {
				return layout, errors.New("exception layout exceptions must be an array")
			}
			for i, raw := range items {
				var item map[string]any
				decoder := json.NewDecoder(bytes.NewReader(raw))
				decoder.UseNumber()
				if decoder.Decode(&item) != nil || item == nil {
					return layout, errors.New("exception layout item invalid")
				}
				id, _ := item["id"].(string)
				tenant, _ := item["tenant_id"].(string)
				if !applicationPlanID(id, 512) || !applicationPlanID(tenant, 64) {
					return layout, errors.New("exception layout identity invalid")
				}
				status, _ := item["status"].(string)
				requestID, _ := item["request_id"].(string)
				if status == "" {
					status = "OPEN"
				}
				if requestID == "" {
					requestID = "-"
				}
				layout.Exceptions = append(layout.Exceptions, exceptionLayoutRow{
					TenantID: tenant, ExceptionID: id, Ordinal: uint64(i) + 1,
					Status: status, RequestID: requestID, Payload: append(json.RawMessage(nil), raw...),
				})
			}
		}
	}
	delete(object, "exceptions")
	var err error
	layout.Root, err = json.Marshal(object)
	return layout, err
}

func joinExceptionLayout(layout exceptionLayout) ([]byte, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(layout.Root, &root) != nil || root == nil {
		return nil, errors.New("invalid exception root envelope")
	}
	if _, ok := root["exceptions"]; ok {
		return nil, errors.New("duplicate exception root authority")
	}
	rows := append([]exceptionLayoutRow(nil), layout.Exceptions...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Ordinal < rows[j].Ordinal })
	items := make([]json.RawMessage, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.ExceptionID] {
			return nil, errors.New("duplicate exception layout row")
		}
		var item map[string]any
		decoder := json.NewDecoder(bytes.NewReader(row.Payload))
		decoder.UseNumber()
		if decoder.Decode(&item) != nil || item == nil {
			return nil, errors.New("exception projection does not match envelope")
		}
		id, _ := item["id"].(string)
		tenant, _ := item["tenant_id"].(string)
		if id != row.ExceptionID || tenant != row.TenantID {
			return nil, errors.New("exception projection identity mismatch")
		}
		seen[row.ExceptionID] = true
		items = append(items, row.Payload)
	}
	switch layout.ExceptionsKind {
	case "absent", "null":
		if len(items) != 0 {
			return nil, errors.New("non-array exception container contains records")
		}
		if layout.ExceptionsKind == "null" {
			root["exceptions"] = json.RawMessage(`null`)
		}
	case "array":
		value, err := json.Marshal(items)
		if err != nil {
			return nil, err
		}
		root["exceptions"] = value
	default:
		return nil, errors.New("unknown exception container shape")
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if _, err = planExceptionSource(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func verifyExceptionLayout(root []byte) error {
	layout, err := splitExceptionLayout(root)
	if err != nil {
		return err
	}
	rebuilt, err := joinExceptionLayout(layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(root, rebuilt) {
		return errors.New("exception layout round-trip reconciliation failed")
	}
	return nil
}