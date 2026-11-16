package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"maps"
	"net/http"
	"strings"
	"time"
)

func settingsStorageKey(p governance.Principal, section string) string {
	owner := ""
	if section == "profile" || section == "general" {
		owner = p.UserID
	}
	raw, _ := json.Marshal([]string{p.TenantID, owner, section})
	return "settings-v1:" + base64.RawURLEncoding.EncodeToString(raw)
}
func settingsValue(st app.State, p governance.Principal, section string) json.RawMessage {
	if value, exists := st.Settings[settingsStorageKey(p, section)]; exists {
		return value
	}
	legacy := settingsLegacyKey(p, section)
	if legacy != "" {
		if value := st.Settings[legacy]; value != nil {
			return value
		}
	}
	return json.RawMessage(`{}`)
}
func settingsLegacyKey(p governance.Principal, section string) string {
	if section == "profile" || section == "general" {
		owner, err := governance.DemoIdentity(p.UserID)
		if err == nil && owner.TenantID == p.TenantID {
			return p.UserID + ":" + section
		}
		return ""
	}
	if p.TenantID == "42" {
		return section
	}
	return ""
}
func settingsRevision(st app.State, p governance.Principal, section string, value json.RawMessage) string {
	key := settingsStorageKey(p, section)
	if revision := st.SettingRevisions[key]; revision != "" {
		return revision
	}
	sum := sha256.Sum256(append([]byte(key+":"), value...))
	return "settings-initial-" + hex.EncodeToString(sum[:])
}
func settingsRecordState(key, legacy string, record app.SettingsRecord) app.State {
	st := app.State{Settings: map[string]json.RawMessage{}, SettingRevisions: map[string]string{key: record.Revision}}
	if record.Canonical {
		st.Settings[key] = record.Value
	} else if legacy != "" {
		st.Settings[legacy] = record.Value
	}
	return st
}
func (s *Server) readSettings(r *http.Request, p governance.Principal, section string) (json.RawMessage, string, error) {
	key, legacy := settingsStorageKey(p, section), settingsLegacyKey(p, section)
	record, handled, e := s.Store.ReadMySQLSettingsContext(r.Context(), key, legacy)
	if e != nil {
		return nil, "", e
	}
	var st app.State
	if handled {
		st = settingsRecordState(key, legacy, record)
	} else {
		st = s.Store.ReadOperational()
	}
	value := settingsValue(st, p, section)
	return value, settingsRevision(st, p, section, value), nil
}
func validateSettings(section string, values map[string]string) error {
	if values == nil {
		return fail(400, "INVALID_REQUEST", "请提供明确的配置对象")
	}
	allowed := map[string]bool{}
	switch section {
	case "profile":
		allowed["display_name"] = true
	case "general":
		allowed = map[string]bool{"language": true, "timezone": true, "density": true}
	case "public":
		allowed["notice"] = true
	}
	for key, value := range values {
		if !allowed[key] || len([]rune(value)) > 512 {
			return fail(400, "INVALID_REQUEST", "包含不属于当前分区或过长的配置项")
		}
		if key == "display_name" {
			value = strings.TrimSpace(value)
			values[key] = value
			if len([]rune(value)) > 80 {
				return fail(400, "INVALID_REQUEST", "显示名称不能超过80个字符")
			}
		}
		if value == "" {
			delete(values, key)
			continue
		}
		switch key {
		case "language":
			if value != "zh-CN" && value != "en-US" {
				return fail(400, "INVALID_REQUEST", "请选择支持的语言偏好")
			}
		case "density":
			if value != "comfortable" && value != "compact" {
				return fail(400, "INVALID_REQUEST", "请选择有效的界面密度偏好")
			}
		case "timezone":
			if _, err := time.LoadLocation(value); err != nil || value == "Local" {
				return fail(400, "INVALID_REQUEST", "请选择明确的IANA时区")
			}
		}
	}
	return nil
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	section := r.PathValue("section")
	if !settingSections[section] {
		return fail(404, "ASSET_UNAVAILABLE", "配置页面不存在")
	}
	if section != "profile" && section != "general" {
		if err := s.requireAdmin(p); err != nil {
			return err
		}
	}
	value, revision, e := s.readSettings(r, p, section)
	if e != nil {
		return e
	}
	Write(w, map[string]any{"section": section, "values": value, "revision": revision, "mode": "validation", "connections_verified": false})
	return nil
}
func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	section := r.PathValue("section")
	if !settingSections[section] {
		return fail(404, "ASSET_UNAVAILABLE", "配置页面不存在")
	}
	if section != "profile" && section != "general" {
		if err := s.requireAdmin(p); err != nil {
			return err
		}
	}
	if section != "profile" && section != "general" && section != "public" {
		return fail(503, "CAPABILITY_UNAVAILABLE", "基础连接尚未接入真实提供方；当前只读，不保存秘密或伪报连接成功")
	}
	var req struct {
		Values   map[string]string `json:"values"`
		Expected *string           `json:"expected_revision,omitempty"`
	}
	if err := Decode(r, &req); err != nil {
		return err
	}
	if err := validateSettings(section, req.Values); err != nil {
		return err
	}
	if req.Expected != nil && len(*req.Expected) > 128 {
		return fail(400, "INVALID_REQUEST", "配置版本标识无效")
	}
	value, _ := json.Marshal(req.Values)
	revision := ""
	update := func(st *app.State) error {
		current := settingsValue(*st, p, section)
		revision = settingsRevision(*st, p, section, current)
		if req.Expected != nil && *req.Expected != revision {
			return fail(409, "VERSION_INCOMPATIBLE", "配置已被其他窗口修改，请重新加载后保存")
		}
		key := settingsStorageKey(p, section)
		if stored, exists := st.Settings[key]; exists {
			var previous map[string]string
			if json.Unmarshal(stored, &previous) == nil && previous != nil && maps.Equal(previous, req.Values) {
				return nil
			}
		}
		st.Settings[key] = value
		revision = app.ID("settings_revision")
		st.SettingRevisions[key] = revision
		appendAudit(st, p, "settings.update", section, "success", "", requestID(r))
		return nil
	}
	key := settingsStorageKey(p, section)
	legacy := settingsLegacyKey(p, section)
	handled, err := s.Store.UpdateMySQLSettingsContext(r.Context(), key, legacy, func(record app.SettingsRecord) (*app.SettingsPatch, error) {
		st := settingsRecordState(key, legacy, record)
		if e := update(&st); e != nil {
			return nil, e
		}
		if len(st.Audit) == 0 {
			return nil, nil
		}
		return &app.SettingsPatch{Value: st.Settings[key], Revision: revision, Audit: st.Audit}, nil
	})
	if !handled && err == nil {
		err = s.Store.UpdateOperationalContext(r.Context(), update)
	}
	if err != nil {
		return err
	}
	Write(w, map[string]any{"values": req.Values, "revision": revision})
	return nil
}

// Display names are presentation only; authorization continues to use Principal(r).
func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	p := Principal(r)
	value, _, e := s.readSettings(r, p, "profile")
	if e != nil {
		return e
	}
	var profile map[string]string
	if json.Unmarshal(value, &profile) == nil {
		if name := strings.TrimSpace(profile["display_name"]); name != "" && len([]rune(name)) <= 80 {
			p.Name = name
		}
	}
	Write(w, p)
	return nil
}
