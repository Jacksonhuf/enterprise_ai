package applications

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"time"
)

// Raw envelopes preserve extensions and JSON numbers; these are not runtime State
// replacements. SQL projections must be checked against them on a future load.
type applicationLayoutRow struct {
	TenantID, AppID string
	Header, Draft   json.RawMessage
	Releases        []json.RawMessage
	ReleasesNull    bool
}
type applicationCheckLayoutRow struct {
	TenantID, AppID, CheckID string
	Payload                  json.RawMessage
}
type applicationLayout struct {
	Root       json.RawMessage
	Apps       []applicationLayoutRow
	Checks     []applicationCheckLayoutRow
	ChecksKind string // absent, null, object
}

func splitApplicationLayout(raw []byte, sampled time.Time) (applicationLayout, error) {
	var layout applicationLayout
	if _, err := planApplicationSource(raw, sampled); err != nil {
		return layout, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return layout, err
	}
	var apps map[string]json.RawMessage
	if err := json.Unmarshal(root["apps"], &apps); err != nil {
		return layout, err
	}
	ids := make([]string, 0, len(apps))
	for id := range apps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	tenants := map[string]string{}
	for _, id := range ids {
		var header map[string]json.RawMessage
		var app App
		if err := json.Unmarshal(apps[id], &header); err != nil {
			return layout, err
		}
		if err := json.Unmarshal(apps[id], &app); err != nil {
			return layout, err
		}
		row := applicationLayoutRow{TenantID: app.TenantID, AppID: id, Draft: header["draft"], ReleasesNull: bytes.Equal(bytes.TrimSpace(header["releases"]), []byte("null"))}
		if err := json.Unmarshal(header["releases"], &row.Releases); err != nil {
			return layout, err
		}
		delete(header, "draft")
		delete(header, "releases")
		var err error
		row.Header, err = json.Marshal(header)
		if err != nil {
			return layout, err
		}
		layout.Apps = append(layout.Apps, row)
		tenants[id] = app.TenantID
	}
	layout.ChecksKind = "absent"
	if checks, exists := root["checks"]; exists {
		layout.ChecksKind = "object"
		if bytes.Equal(bytes.TrimSpace(checks), []byte("null")) {
			layout.ChecksKind = "null"
		}
		var objects map[string]json.RawMessage
		if err := json.Unmarshal(checks, &objects); err != nil {
			return layout, err
		}
		ids = ids[:0]
		for id := range objects {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			var c Check
			if err := json.Unmarshal(objects[id], &c); err != nil {
				return layout, err
			}
			layout.Checks = append(layout.Checks, applicationCheckLayoutRow{tenants[c.AppID], c.AppID, id, objects[id]})
		}
	}
	delete(root, "apps")
	delete(root, "checks")
	var err error
	layout.Root, err = json.Marshal(root)
	return layout, err
}

func joinApplicationLayout(layout applicationLayout, sampled time.Time) ([]byte, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(layout.Root, &root) != nil || root == nil {
		return nil, errors.New("invalid application root envelope")
	}
	if _, ok := root["apps"]; ok {
		return nil, errors.New("duplicate application root authority")
	}
	if _, ok := root["checks"]; ok {
		return nil, errors.New("duplicate check root authority")
	}
	apps := map[string]json.RawMessage{}
	tenants := map[string]string{}
	for _, row := range layout.Apps {
		var header map[string]json.RawMessage
		var identity App
		if json.Unmarshal(row.Header, &header) != nil || header == nil || json.Unmarshal(row.Header, &identity) != nil || identity.ID != row.AppID || identity.TenantID != row.TenantID {
			return nil, errors.New("application projection does not match envelope")
		}
		if _, exists := apps[row.AppID]; exists {
			return nil, errors.New("duplicate application layout row")
		}
		if _, exists := header["draft"]; exists {
			return nil, errors.New("duplicate draft authority")
		}
		if _, exists := header["releases"]; exists {
			return nil, errors.New("duplicate release authority")
		}
		header["draft"] = row.Draft
		if row.ReleasesNull {
			if len(row.Releases) != 0 {
				return nil, errors.New("null release container contains records")
			}
			header["releases"] = json.RawMessage(`null`)
		} else {
			releases := row.Releases
			if releases == nil {
				releases = []json.RawMessage{}
			}
			value, err := json.Marshal(releases)
			if err != nil {
				return nil, err
			}
			header["releases"] = value
		}
		value, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}
		apps[row.AppID] = value
		tenants[row.AppID] = row.TenantID
	}
	checks := map[string]json.RawMessage{}
	for _, row := range layout.Checks {
		var check Check
		tenant, exists := tenants[row.AppID]
		if json.Unmarshal(row.Payload, &check) != nil || !exists || tenant != row.TenantID || check.ID != row.CheckID || check.AppID != row.AppID {
			return nil, errors.New("check projection does not match parent or envelope")
		}
		if _, exists := checks[row.CheckID]; exists {
			return nil, errors.New("duplicate check layout row")
		}
		checks[row.CheckID] = row.Payload
	}
	value, err := json.Marshal(apps)
	if err != nil {
		return nil, err
	}
	root["apps"] = value
	switch layout.ChecksKind {
	case "absent", "null":
		if len(checks) != 0 {
			return nil, errors.New("non-object check container contains records")
		}
		if layout.ChecksKind == "null" {
			root["checks"] = json.RawMessage(`null`)
		}
	case "object":
		value, err = json.Marshal(checks)
		if err != nil {
			return nil, err
		}
		root["checks"] = value
	default:
		return nil, errors.New("unknown check container shape")
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if _, err = planApplicationSource(raw, sampled); err != nil {
		return nil, err
	}
	return raw, nil
}

// UseNumber prevents an integrity check from rounding distinct large integers to
// the same float64. Object ordering/spacing may change; values and shapes may not.
func equalApplicationJSON(left, right []byte) bool {
	if !json.Valid(left) || !json.Valid(right) {
		return false
	}
	decode := func(raw []byte) (any, error) {
		var value any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		err := d.Decode(&value)
		return value, err
	}
	a, err := decode(left)
	if err != nil {
		return false
	}
	b, err := decode(right)
	return err == nil && reflect.DeepEqual(a, b)
}
func verifyApplicationLayout(raw []byte, sampled time.Time) error {
	layout, err := splitApplicationLayout(raw, sampled)
	if err != nil {
		return err
	}
	rebuilt, err := joinApplicationLayout(layout, sampled)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(raw, rebuilt) {
		return errors.New("application layout round-trip reconciliation failed")
	}
	return nil
}
