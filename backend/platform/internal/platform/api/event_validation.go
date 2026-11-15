package api

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func eventFields(object map[string]any, required, optional []string) bool {
	if object == nil {
		return false
	}
	allowed := map[string]bool{}
	for _, k := range required {
		allowed[k] = true
		if _, ok := object[k]; !ok {
			return false
		}
	}
	for _, k := range optional {
		allowed[k] = true
	}
	for k := range object {
		if !allowed[k] {
			return false
		}
	}
	return true
}
func eventString(value any, max int) bool {
	s, ok := value.(string)
	return ok && utf8.RuneCountInString(s) > 0 && utf8.RuneCountInString(s) <= max
}
func eventNullableID(value any, max int) bool { return value == nil || eventString(value, max) }
func eventTenant(value any) bool {
	s, ok := value.(string)
	if !ok || s == "" || s[0] < '1' || s[0] > '9' {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}

var eventDateTime = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}[Tt][0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?([Zz]|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

func validEvent(kind string, e map[string]any) bool {
	if !eventString(e["event_id"], 128) {
		return false
	}
	stamp, ok := e["occurred_at"].(string)
	if !ok {
		return false
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.ToUpper(stamp))
	if err != nil || parsed.Year() < 1 || !eventDateTime.MatchString(stamp) {
		return false
	}
	producer, ok := e["producer"].(string)
	if !ok || len(producer) != 1 || producer < "A" || producer > "G" {
		return false
	}
	if kind == "usage-events" {
		if producer == "G" || !eventFields(e, []string{"event_id", "occurred_at", "producer", "tenant_id", "capability", "count", "unit"}, []string{"application_id", "execution_id"}) || !eventTenant(e["tenant_id"]) || !eventString(e["capability"], 64) {
			return false
		}
		count, ok := e["count"].(float64)
		if !ok || count < 0 {
			return false
		}
		if e["unit"] != "CALL" && e["unit"] != "TOKEN" && e["unit"] != "MILLISECOND" {
			return false
		}
		for _, key := range []string{"application_id", "execution_id"} {
			if value, present := e[key]; present && !eventString(value, 128) {
				return false
			}
		}
		return true
	}
	if kind != "audit-events" || !eventFields(e, []string{"event_id", "event_type", "occurred_at", "producer", "subject", "resource", "action", "outcome", "correlation"}, []string{"evidence_ref"}) || !eventString(e["event_type"], 64) || !eventString(e["action"], 64) {
		return false
	}
	subject, _ := e["subject"].(map[string]any)
	if !eventFields(subject, []string{"tenant_id", "actor_user_id", "client_id"}, nil) || !eventTenant(subject["tenant_id"]) || !eventNullableID(subject["actor_user_id"], 64) || !eventNullableID(subject["client_id"], 128) {
		return false
	}
	resource, _ := e["resource"].(map[string]any)
	if !eventFields(resource, []string{"type", "id"}, nil) || !eventString(resource["type"], 32) || !eventString(resource["id"], 64) {
		return false
	}
	correlation, _ := e["correlation"].(map[string]any)
	if !eventFields(correlation, nil, []string{"turn_id", "execution_id", "service_execution_id"}) {
		return false
	}
	for _, value := range correlation {
		if !eventNullableID(value, 128) {
			return false
		}
	}
	if value, present := e["evidence_ref"]; present && value != nil {
		ref, _ := value.(map[string]any)
		if !eventFields(ref, []string{"execution_id"}, nil) || !eventString(ref["execution_id"], 128) {
			return false
		}
	}
	switch e["outcome"] {
	case "success", "accepted", "denied", "failed", "partial", "canceled":
		return true
	}
	return false
}
