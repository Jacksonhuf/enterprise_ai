package api_test

import (
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func sessionMeta(t *testing.T, f *fixture, id string) map[string]any {
	return decode(t, f.request("demo-user", "GET", "/api/v1/sessions/"+id+"?include_messages=false", nil, 200))["session"].(map[string]any)
}
func rename(t *testing.T, f *fixture, id, revision, title string, status int) map[string]any {
	return decode(t, f.request("demo-user", "PUT", "/api/v1/sessions/"+id, map[string]string{"title": title, "expected_revision": revision}, status))
}
func TestSessionRenameCASPersistenceAndIsolation(t *testing.T) {
	f := setup(t)
	id := f.session()
	v := sessionMeta(t, f, id)
	rev := v["revision"].(string)
	rename(t, f, id, rev, "  我的分析  ", 200)
	v = sessionMeta(t, f, id)
	rev1 := v["revision"].(string)
	if v["title"] != "我的分析" || rev == rev1 {
		t.Fatal(v)
	}
	rename(t, f, id, rev, "stale", 409)
	rename(t, f, id, rev1, "我的分析", 200)
	count := 0
	for _, e := range f.store.Read().Audit {
		if e.Action == "session.rename" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("no-op audited", count)
	}
	rename(t, f, id, rev1, "第二版", 200)
	v = sessionMeta(t, f, id)
	rename(t, f, id, v["revision"].(string), "我的分析", 200)
	rename(t, f, id, rev1, "ABA", 409)
	for _, title := range []string{" ", strings.Repeat("字", 81)} {
		rename(t, f, id, sessionMeta(t, f, id)["revision"].(string), title, 400)
	}
	f.request("demo-admin", "PUT", "/api/v1/sessions/"+id, map[string]string{"title": "cross-owner", "expected_revision": rev1}, 404)
	f.request("demo-outsider", "PUT", "/api/v1/sessions/"+id, map[string]string{"title": "cross-tenant", "expected_revision": rev1}, 404)
	// A first completed turn must preserve the user's title while updating activity.
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]any{"query": "分析SO1002"}, 200)
	if sessionMeta(t, f, id)["title"] != "我的分析" {
		t.Fatal("chat overwrote custom title")
	}
	path := filepath.Join(t.TempDir(), "saved.json")
	saved, e := app.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	state := f.store.Read()
	if e = saved.Update(func(st *app.State) error { *st = state; return nil }); e != nil {
		t.Fatal(e)
	}
	reopened, e := app.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if reopened.Read().Sessions[id].Title != "我的分析" {
		t.Fatal("lost persisted rename")
	}
}
func TestSessionListSearchPaginationAndLegacyActivity(t *testing.T) {
	f := setup(t)
	id := f.session()
	sample := f.store.Read().Sessions[id]
	if e := f.store.Update(func(st *app.State) error {
		delete(st.Sessions, id)
		for i := 0; i < 45; i++ {
			v := sample
			v.ID = fmt.Sprintf("legacy-%02d", i)
			v.Title = fmt.Sprintf("分析-%02d", i)
			v.CreatedAt = ""
			v.UpdatedAt = ""
			v.Revision = ""
			v.Messages = []app.Message{{CreatedAt: fmt.Sprintf("2026-11-15t12:%02d:00z", i)}}
			st.Sessions[v.ID] = v
		}
		v := sample
		v.ID = "foreign"
		v.OwnerID = "someone-else"
		v.Title = "机密"
		st.Sessions[v.ID] = v
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	base := "/api/v1/sessions?app_id=" + url.QueryEscape(sample.AppID) + "&limit=17"
	page := decode(t, f.request("demo-user", "GET", base, nil, 200))
	rows := page["items"].([]any)
	if len(rows) != 17 || rows[0].(map[string]any)["session_id"] != "legacy-44" || rows[0].(map[string]any)["created_at"] != nil || rows[0].(map[string]any)["messages"] != nil {
		t.Fatal("invalid metadata/order", page)
	}
	cursor := page["next_cursor"].(string)
	seen := map[string]bool{}
	for {
		for _, row := range page["items"].([]any) {
			key := row.(map[string]any)["session_id"].(string)
			if seen[key] {
				t.Fatal("duplicate", key)
			}
			seen[key] = true
		}
		if page["next_cursor"] == nil {
			break
		}
		page = decode(t, f.request("demo-user", "GET", base+"&cursor="+url.QueryEscape(page["next_cursor"].(string)), nil, 200))
	}
	if len(seen) != 45 || seen["foreign"] {
		t.Fatal("missing or leaked", len(seen))
	}
	f.request("demo-user", "GET", base+"&query=other&cursor="+url.QueryEscape(cursor), nil, 400)
	for _, suffix := range []string{"&limit=0", "&limit=1.5", "&limit=101", "&cursor=invalid", "&query=" + url.QueryEscape(strings.Repeat("字", 257))} {
		f.request("demo-user", "GET", base+suffix, nil, 400)
	}
	filtered := decode(t, f.request("demo-user", "GET", base+"&query="+url.QueryEscape(" 分析-44 "), nil, 200))
	if len(filtered["items"].([]any)) != 1 {
		t.Fatal(filtered)
	}
	legacy := sessionMeta(t, f, "legacy-00")
	rename(t, f, "legacy-00", legacy["revision"].(string), "新名字", 200)
	f.request("demo-user", "GET", base+"&cursor="+url.QueryEscape(cursor), nil, 400)
}
