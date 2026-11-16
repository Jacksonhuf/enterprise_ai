package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var pagedGovernanceSections = map[string]bool{"audit": true, "exceptions": true, "usage": true, "grants": true}

type governanceCursor struct {
	Version int    `json:"v"`
	After   string `json:"after"`
	Filter  string `json:"filter"`
}

func governanceRowID(row map[string]any) string {
	for _, key := range []string{"event_id", "grant_id", "id"} {
		if id, ok := row[key].(string); ok && id != "" {
			return id
		}
	}
	return ""
}
func governancePage(items any, r *http.Request, tenant, section string) (map[string]any, error) {
	q := r.URL.Query()
	query := strings.ToLower(strings.TrimSpace(q.Get("query")))
	if len([]rune(query)) > 256 {
		return nil, fail(400, "INVALID_REQUEST", "搜索内容不能超过256个字符")
	}
	limit := 20
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return nil, fail(400, "INVALID_REQUEST", "每页数量应为1到100")
		}
		limit = n
	}
	status, outcome, unit := q.Get("status"), q.Get("outcome"), q.Get("unit")
	allowedStatus := map[string]bool{}
	if section == "exceptions" {
		allowedStatus = map[string]bool{"OPEN": true, "ACKNOWLEDGED": true, "RESOLVED": true}
	}
	if section == "grants" {
		allowedStatus = map[string]bool{"ACTIVE": true, "REVOKED": true}
	}
	if status != "" && !allowedStatus[status] {
		return nil, fail(400, "INVALID_REQUEST", "状态筛选不适用于当前列表")
	}
	if outcome != "" && (section != "audit" || !map[string]bool{"success": true, "accepted": true, "denied": true, "failed": true, "partial": true, "canceled": true}[outcome]) {
		return nil, fail(400, "INVALID_REQUEST", "审计结果筛选无效")
	}
	if unit != "" && (section != "usage" || !map[string]bool{"CALL": true, "TOKEN": true, "MILLISECOND": true}[unit]) {
		return nil, fail(400, "INVALID_REQUEST", "用量单位筛选无效")
	}
	var from, to time.Time
	hasFrom, hasTo := q.Get("from_time") != "", q.Get("to_time") != ""
	for _, field := range []string{"from_time", "to_time"} {
		if value := q.Get(field); value != "" {
			stamp, err := time.Parse(time.RFC3339Nano, strings.ToUpper(value))
			if err != nil || stamp.Year() < 1 || section == "grants" {
				return nil, fail(400, "INVALID_REQUEST", "时间范围无效或不适用于当前列表")
			}
			if field == "from_time" {
				from = stamp
			} else {
				to = stamp
			}
		}
	}
	if hasFrom && hasTo && from.After(to) {
		return nil, fail(400, "INVALID_REQUEST", "开始时间不能晚于结束时间")
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	rows := []map[string]any{}
	if err = json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if section == "grants" || section == "usage" {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
	filtered := []map[string]any{}
	searchFields := []string{"event_id", "grant_id", "id", "name", "actor_user_id", "client_id", "application_id", "subject_id", "resource_id", "action", "request_id", "execution_id", "capability", "code", "route", "module"}
	for _, row := range rows {
		if query != "" {
			found := false
			for _, field := range searchFields {
				value, _ := row[field].(string)
				if strings.Contains(strings.ToLower(value), query) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if status != "" {
			if section == "grants" {
				active, _ := row["active"].(bool)
				if active != (status == "ACTIVE") {
					continue
				}
			} else if row["status"] != status {
				continue
			}
		}
		if outcome != "" && row["outcome"] != outcome {
			continue
		}
		if unit != "" && row["unit"] != unit {
			continue
		}
		if hasFrom || hasTo {
			value, _ := row["occurred_at"].(string)
			stamp, e := time.Parse(time.RFC3339Nano, strings.ToUpper(value))
			if e != nil || (hasFrom && stamp.Before(from)) || (hasTo && stamp.After(to)) {
				continue
			}
		}
		filtered = append(filtered, row)
	}
	// A cursor locates an authorized row. Its fingerprint binds it to tenant/section/filters,
	// not to a cached authorization decision; all rows are filtered again on every request.
	stamp := func(t time.Time, present bool) string {
		if !present {
			return ""
		}
		return t.UTC().Format(time.RFC3339Nano)
	}
	fingerprint, _ := json.Marshal([]string{tenant, section, query, status, outcome, unit, stamp(from, hasFrom), stamp(to, hasTo)})
	hash := sha256.Sum256(fingerprint)
	digest := hex.EncodeToString(hash[:])
	start := 0
	if cursor := q.Get("cursor"); cursor != "" {
		if len(cursor) > 2048 {
			return nil, fail(400, "INVALID_REQUEST", "分页游标无效")
		}
		decoded, e := base64.RawURLEncoding.DecodeString(cursor)
		var c governanceCursor
		if e != nil || json.Unmarshal(decoded, &c) != nil || c.Version != 1 || c.After == "" || c.Filter != digest {
			return nil, fail(400, "INVALID_REQUEST", "分页筛选已变化，请刷新列表")
		}
		start = -1
		for i, row := range filtered {
			if governanceRowID(row) == c.After {
				start = i + 1
				break
			}
		}
		if start < 0 {
			return nil, fail(400, "INVALID_REQUEST", "分页记录已变化，请刷新列表")
		}
	}
	end := min(start+limit, len(filtered))
	var next any
	if end < len(filtered) {
		id := governanceRowID(filtered[end-1])
		if id == "" {
			return nil, fail(503, "CAPABILITY_UNAVAILABLE", "记录缺少稳定标识，无法继续分页")
		}
		b, _ := json.Marshal(governanceCursor{1, id, digest})
		next = base64.RawURLEncoding.EncodeToString(b)
	}
	return map[string]any{"items": filtered[start:end], "next_cursor": next, "sampled_at": time.Now().UTC().Format(time.RFC3339), "mode": "validation"}, nil
}
