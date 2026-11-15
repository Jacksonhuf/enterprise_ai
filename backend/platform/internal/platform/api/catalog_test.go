package api_test

import (
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"net/url"
	"strings"
	"testing"
)

func TestUseSearchUsesPublishedMetadataOnly(t *testing.T) {
	f := setup(t)
	err := f.store.Update(func(st *app.State) error {
		a := st.Apps["channel-replenishment"]
		a.Draft.Basic.Name = "unpublished-secret-name"
		a.Draft.Basic.Description = "unpublished-secret-description"
		st.Apps[a.ID] = a
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"unpublished-secret-name", "unpublished-secret-description"} {
		response := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/apps?purpose=USE&query="+url.QueryEscape(q), nil, 200))
		if len(response["items"].([]any)) != 0 {
			t.Fatal("unpublished metadata affected user search")
		}
	}
	a := f.store.Read().Apps["channel-replenishment"]
	rel, _ := app.Current(a)
	response := decode(t, f.request("demo-user", "GET", "/api/v1/cognitive/apps?purpose=USE&query="+url.QueryEscape(rel.Config.Basic.Name), nil, 200))
	if len(response["items"].([]any)) != 1 {
		t.Fatal("published name was not searchable")
	}
	managed := decode(t, f.request("demo-engineer", "GET", "/api/v1/cognitive/apps?purpose=MANAGE&query=unpublished-secret-name", nil, 200))
	if len(managed["items"].([]any)) != 1 {
		t.Fatal("builder cannot search draft")
	}
}
func TestCatalogAndReleasePaginationStableAndIsolated(t *testing.T) {
	f := setup(t)
	err := f.store.Update(func(st *app.State) error {
		a := st.Apps["channel-replenishment"]
		rel, _ := app.Current(a)
		for i := 0; i < 25; i++ {
			copy := a
			copy.ID = fmt.Sprintf("paged-%02d", i)
			copy.Draft.Basic.Name = copy.ID
			st.Apps[copy.ID] = copy
		}
		for i := 0; i < 25; i++ {
			copy := rel
			copy.ID = fmt.Sprintf("release-paged-%02d", i)
			copy.PublishedAt = "2026-11-15T00:00:00Z"
			a.Releases = append(a.Releases, copy)
		}
		st.Apps[a.ID] = a
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"/api/v1/cognitive/apps?purpose=MANAGE&query=paged-", "/api/v1/cognitive/apps/channel-replenishment/releases?"} {
		seen := map[string]bool{}
		previousRelease := ""
		cursor := ""
		for {
			response := decode(t, f.request("demo-engineer", "GET", endpoint+"&limit=7&cursor="+cursor, nil, 200))
			for _, row := range response["items"].([]any) {
				value := row.(map[string]any)
				key, _ := value["release_id"].(string)
				if key == "" {
					key = value["app_id"].(string)
				}
				if seen[key] {
					t.Fatal("duplicate pagination row", key)
				}
				seen[key] = true
				if strings.HasPrefix(key, "release-paged-") {
					if previousRelease != "" && previousRelease <= key {
						t.Fatal("equal-time releases not sorted consistently", previousRelease, key)
					}
					previousRelease = key
				}
			}
			next, ok := response["next_cursor"].(string)
			if !ok {
				break
			}
			if next == cursor {
				t.Fatal("cursor not advancing")
			}
			cursor = next
		}
		if len(seen) < 25 {
			t.Fatalf("truncated rows %d", len(seen))
		}
	}
	out := decode(t, f.request("demo-outsider", "GET", "/api/v1/cognitive/apps?purpose=USE", nil, 200))
	if len(out["items"].([]any)) != 0 {
		t.Fatal("cross-tenant catalog leaked")
	}
	f.request("demo-user", "GET", "/api/v1/cognitive/apps/channel-replenishment/releases", nil, 403)
}
