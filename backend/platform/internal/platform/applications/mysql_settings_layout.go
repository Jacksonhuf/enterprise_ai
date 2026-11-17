package applications

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

type settingsLayoutRow struct {
	TenantID   string          `json:"tenant_id"`
	SettingKey string          `json:"setting_key"`
	OwnerID    string          `json:"owner_id"`
	Section    string          `json:"section"`
	Revision   string          `json:"revision"`
	Payload    json.RawMessage `json:"payload_raw"`
}

type settingsLayout struct {
	Root               json.RawMessage `json:"root_payload"`
	Settings           []settingsLayoutRow
	SettingsKind       string
	RevisionsKind      string
	InferredTenant     string
}

func parseSettingKey(key string) (owner, section string, err error) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 || !applicationPlanID(parts[0], 512) || !applicationPlanID(parts[1], 128) {
		return "", "", errors.New("invalid setting key")
	}
	return parts[0], parts[1], nil
}

func splitSettingsLayout(root []byte) (settingsLayout, error) {
	var layout settingsLayout
	var object map[string]json.RawMessage
	if json.Unmarshal(root, &object) != nil || object == nil {
		return layout, errors.New("settings layout root invalid")
	}
	for _, key := range []string{"apps", "checks", "grants", "sessions"} {
		if _, ok := object[key]; ok {
			return layout, errors.New("settings layout requires schema5 root")
		}
	}
	layout.SettingsKind, layout.RevisionsKind = "absent", "absent"
	if value, ok := object["settings"]; ok {
		layout.SettingsKind = "object"
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			layout.SettingsKind = "null"
		}
	}
	if value, ok := object["setting_revisions"]; ok {
		layout.RevisionsKind = "object"
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			layout.RevisionsKind = "null"
		}
	}
	if layout.SettingsKind == "null" || layout.RevisionsKind == "null" {
		return layout, errors.New("settings layout null container unsupported")
	}
	if layout.SettingsKind == "object" {
		var items map[string]json.RawMessage
		settingsRaw, _ := object["settings"]
		if json.Unmarshal(settingsRaw, &items) != nil || items == nil {
			return layout, errors.New("settings layout settings must be an object")
		}
		var revisions map[string]string
		if layout.RevisionsKind == "object" {
			revRaw := object["setting_revisions"]
			if json.Unmarshal(revRaw, &revisions) != nil || revisions == nil {
				return layout, errors.New("settings layout setting_revisions must be an object")
			}
		} else if len(items) != 0 {
			return layout, errors.New("settings layout missing revision map")
		}
		keys := make([]string, 0, len(items))
		for key := range items {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			owner, section, err := parseSettingKey(key)
			if err != nil {
				return layout, err
			}
			rev, ok := revisions[key]
			if !ok || !applicationPlanID(rev, 512) {
				return layout, errors.New("settings layout revision missing or invalid")
			}
			layout.Settings = append(layout.Settings, settingsLayoutRow{
				SettingKey: key, OwnerID: owner, Section: section, Revision: rev, Payload: append(json.RawMessage(nil), items[key]...),
			})
		}
		for key := range revisions {
			if _, ok := items[key]; !ok {
				return layout, errors.New("settings layout orphan revision")
			}
		}
	} else if layout.RevisionsKind == "object" {
		var revisions map[string]string
		if json.Unmarshal(object["setting_revisions"], &revisions) != nil || len(revisions) != 0 {
			return layout, errors.New("settings layout orphan revision container")
		}
	}
	delete(object, "settings")
	delete(object, "setting_revisions")
	var err error
	layout.Root, err = json.Marshal(object)
	return layout, err
}

func joinSettingsLayout(layout settingsLayout) ([]byte, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(layout.Root, &root) != nil || root == nil {
		return nil, errors.New("invalid settings root envelope")
	}
	if _, ok := root["settings"]; ok {
		return nil, errors.New("duplicate settings root authority")
	}
	if _, ok := root["setting_revisions"]; ok {
		return nil, errors.New("duplicate settings root authority")
	}
	settings := map[string]json.RawMessage{}
	revisions := map[string]string{}
	seen := map[string]bool{}
	tenant := layout.InferredTenant
	for _, row := range layout.Settings {
		if seen[row.SettingKey] {
			return nil, errors.New("duplicate settings layout row")
		}
		seen[row.SettingKey] = true
		if tenant == "" {
			tenant = row.TenantID
		}
		if row.TenantID != "" && tenant != row.TenantID {
			return nil, errors.New("settings layout tenant mismatch")
		}
		owner, section, err := parseSettingKey(row.SettingKey)
		if err != nil || owner != row.OwnerID || section != row.Section {
			return nil, errors.New("settings projection does not match envelope")
		}
		settings[row.SettingKey] = append(json.RawMessage(nil), row.Payload...)
		revisions[row.SettingKey] = row.Revision
	}
	switch layout.SettingsKind {
	case "absent":
		if len(settings) != 0 {
			return nil, errors.New("non-object settings container contains records")
		}
	case "object":
		value, err := json.Marshal(settings)
		if err != nil {
			return nil, err
		}
		root["settings"] = value
	default:
		return nil, errors.New("unknown settings container shape")
	}
	switch layout.RevisionsKind {
	case "absent":
		if len(revisions) != 0 {
			return nil, errors.New("non-object revision container contains records")
		}
	case "object":
		value, err := json.Marshal(revisions)
		if err != nil {
			return nil, err
		}
		root["setting_revisions"] = value
	default:
		return nil, errors.New("unknown setting_revisions container shape")
	}
	raw, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	if _, err = planSettingsSource(raw, layout.InferredTenant); err != nil {
		return nil, err
	}
	if tenant == "" && len(layout.Settings) > 0 {
		return nil, errors.New("settings layout requires tenant context")
	}
	return raw, nil
}

func verifySettingsLayout(root []byte, tenant string) error {
	layout, err := splitSettingsLayout(root)
	if err != nil {
		return err
	}
	layout.InferredTenant = tenant
	for i := range layout.Settings {
		layout.Settings[i].TenantID = tenant
	}
	rebuilt, err := joinSettingsLayout(layout)
	if err != nil {
		return err
	}
	if !equalApplicationJSON(root, rebuilt) {
		return errors.New("settings layout round-trip reconciliation failed")
	}
	return nil
}

func dominantTenantID(st State) (string, error) {
	if len(st.Apps) == 0 {
		return "", errors.New("settings plan requires application tenant context")
	}
	var tenant string
	for _, a := range st.Apps {
		if tenant == "" {
			tenant = a.TenantID
		}
		if a.TenantID != tenant {
			return "", errors.New("settings plan ambiguous tenant")
		}
	}
	return tenant, nil
}