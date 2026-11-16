package api_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRestoreHistoricalReleaseDraftRequiresFreshCheck(t *testing.T) {
	f := setup(t)
	original := f.store.Read().Apps["channel-replenishment"]
	path := "/api/v1/cognitive/apps/" + original.ID
	historical := original.Releases[0]
	save := func(config any, revision string) map[string]any {
		data, _ := json.Marshal(config)
		var payload map[string]any
		json.Unmarshal(data, &payload)
		payload["expected_draft_revision"] = revision
		return decode(t, f.request("demo-engineer", "PUT", path, payload, 200))
	}
	check := func(revision any) map[string]any {
		result := decode(t, f.request("demo-engineer", "POST", path+"/release-checks", map[string]any{"expected_draft_revision": revision}, 200))
		if result["status"] != "PASS" {
			t.Fatal(result)
		}
		return result
	}
	publish := func(c map[string]any, revision any, current any, status int) []byte {
		return f.request("demo-engineer", "POST", path+"/releases", map[string]any{"check_id": c["check_id"], "expected_draft_revision": revision, "expected_current_release_id": current, "release_note": "恢复验证"}, status)
	}
	modified := original.Draft
	modified.Basic.Name = "Updated before restoration"
	saved := save(modified, original.Revision)
	published := decode(t, publish(check(saved["draft_revision"]), saved["draft_revision"], original.CurrentReleaseID, 200))
	live := published["release_id"]
	staleCheck := check(saved["draft_revision"])
	restored := save(historical.Config, saved["draft_revision"].(string))
	now := f.store.Read().Apps[original.ID]
	if *now.CurrentReleaseID != live || !reflect.DeepEqual(now.Draft, historical.Config) || now.Revision == saved["draft_revision"] {
		t.Fatal("restore changed live pointer or lost draft CAS")
	}
	if !reflect.DeepEqual(now.Releases[0], historical) || len(now.Releases) != len(original.Releases)+1 {
		t.Fatal("restore mutated release history")
	}
	data, _ := json.Marshal(modified)
	var stale map[string]any
	json.Unmarshal(data, &stale)
	stale["expected_draft_revision"] = saved["draft_revision"]
	f.request("demo-engineer", "PUT", path, stale, 409)
	publish(staleCheck, restored["draft_revision"], live, 409)
	fresh := check(restored["draft_revision"])
	next := decode(t, publish(fresh, restored["draft_revision"], live, 200))
	if next["release_id"] == live {
		t.Fatal("republish reused release")
	}
	final := f.store.Read().Apps[original.ID]
	if !reflect.DeepEqual(final.Releases[len(final.Releases)-1].Config, historical.Config) {
		t.Fatal("republish lost restored config")
	}
}
