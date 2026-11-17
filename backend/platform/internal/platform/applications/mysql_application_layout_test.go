package applications

import (
	"encoding/json"
	"testing"
	"time"
)

func rawLayoutFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(applicationPlanFixture())
	if err != nil {
		t.Fatal(err)
	}
	var root, apps, app, draft map[string]json.RawMessage
	json.Unmarshal(raw, &root)
	json.Unmarshal(root["apps"], &apps)
	json.Unmarshal(apps["channel-replenishment"], &app)
	json.Unmarshal(app["draft"], &draft)
	extension := json.RawMessage(`{"integer":9007199254740993,"decimal":0.1234567890123456789,"exponent":1e25,"null":null,"empty":[],"text":"企业 <notice>"}`)
	root["future"] = extension
	app["future_header"] = extension
	draft["future_config"] = extension
	app["draft"], _ = json.Marshal(draft)
	var releases []map[string]json.RawMessage
	json.Unmarshal(app["releases"], &releases)
	releases[0]["future_release"] = extension
	var snapshot map[string]json.RawMessage
	json.Unmarshal(releases[0]["config_snapshot"], &snapshot)
	snapshot["future_snapshot"] = extension
	releases[0]["config_snapshot"], _ = json.Marshal(snapshot)
	second := map[string]json.RawMessage{}
	for key, value := range releases[0] {
		second[key] = value
	}
	second["release_id"] = json.RawMessage(`"second-release"`)
	releases = append(releases, second)
	app["releases"], _ = json.Marshal(releases)
	var checks map[string]map[string]json.RawMessage
	json.Unmarshal(root["checks"], &checks)
	checks["old-check"]["future_check"] = extension
	root["checks"], _ = json.Marshal(checks)
	apps["channel-replenishment"], _ = json.Marshal(app)
	root["apps"], _ = json.Marshal(apps)
	raw, err = json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func TestApplicationLayoutRoundTripPreservesOpaqueValuesAndShapes(t *testing.T) {
	raw := rawLayoutFixture(t)
	before := string(raw)
	if err := verifyApplicationLayout(raw, time.Now()); err != nil {
		t.Fatal(err)
	}
	if string(raw) != before {
		t.Fatal("source changed")
	}
	// Root check and per-application release collection shapes are independent.
	for _, kind := range []string{"absent", "null", "object"} {
		for _, releases := range []string{"null", "[]"} {
			var root, apps, app map[string]json.RawMessage
			json.Unmarshal(raw, &root)
			json.Unmarshal(root["apps"], &apps)
			json.Unmarshal(apps["channel-replenishment"], &app)
			app["releases"] = json.RawMessage(releases)
			app["current_release_id"] = json.RawMessage("null")
			apps["channel-replenishment"], _ = json.Marshal(app)
			root["apps"], _ = json.Marshal(apps)
			switch kind {
			case "absent":
				delete(root, "checks")
			case "null":
				root["checks"] = json.RawMessage("null")
			case "object":
				root["checks"] = json.RawMessage("{}")
			}
			value, _ := json.Marshal(root)
			if err := verifyApplicationLayout(value, time.Now()); err != nil {
				t.Fatal(kind, releases, err)
			}
		}
	}
}
func TestApplicationLayoutRejectsTamperedProjectionsAndLostRecords(t *testing.T) {
	raw := rawLayoutFixture(t)
	layout, err := splitApplicationLayout(raw, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*applicationLayout){
		"app-tenant":          func(l *applicationLayout) { l.Apps[0].TenantID = "foreign" },
		"app-id":              func(l *applicationLayout) { l.Apps[0].AppID = "other" },
		"duplicate-app":       func(l *applicationLayout) { l.Apps = append(l.Apps, l.Apps[0]) },
		"check-tenant":        func(l *applicationLayout) { l.Checks[0].TenantID = "foreign" },
		"check-id":            func(l *applicationLayout) { l.Checks[0].CheckID = "other" },
		"duplicate-check":     func(l *applicationLayout) { l.Checks = append(l.Checks, l.Checks[0]) },
		"release-shape":       func(l *applicationLayout) { l.Apps[0].ReleasesNull = true },
		"check-shape":         func(l *applicationLayout) { l.ChecksKind = "absent" },
		"duplicate-authority": func(l *applicationLayout) { l.Root = json.RawMessage(`{"apps":{}}`) },
		"invalid-kind":        func(l *applicationLayout) { l.ChecksKind = "unknown" },
	} {
		t.Run(name, func(t *testing.T) {
			copyRaw, _ := json.Marshal(layout)
			var copy applicationLayout
			json.Unmarshal(copyRaw, &copy)
			mutate(&copy)
			if _, err := joinApplicationLayout(copy, time.Now()); err == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
	layout.Apps[0].Releases[0], layout.Apps[0].Releases[1] = layout.Apps[0].Releases[1], layout.Apps[0].Releases[0]
	reordered, err := joinApplicationLayout(layout, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if equalApplicationJSON(raw, reordered) {
		t.Fatal("release order change escaped reconciliation")
	}
	layout.Apps[0].Releases[0], layout.Apps[0].Releases[1] = layout.Apps[0].Releases[1], layout.Apps[0].Releases[0]
	layout.Checks = nil
	joined, err := joinApplicationLayout(layout, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if equalApplicationJSON(raw, joined) {
		t.Fatal("missing record escaped reconciliation")
	}
}
func TestApplicationJSONReconciliationDoesNotRoundNumbers(t *testing.T) {
	for _, pair := range [][2]string{{`9007199254740993`, `9007199254740992`}, {`null`, `[]`}, {`{}`, `[]`}, {`1 2`, `1`}, {`{"x":0.1234567890123456789}`, `{"x":0.1234567890123456788}`}} {
		if equalApplicationJSON([]byte(pair[0]), []byte(pair[1])) {
			t.Fatal("distinct JSON values treated as equal", pair)
		}
	}
	if !equalApplicationJSON([]byte(`{"a":9007199254740993,"b":[]}`), []byte(`{ "b": [], "a":9007199254740993 }`)) {
		t.Fatal("ordering/whitespace rejected")
	}
}
