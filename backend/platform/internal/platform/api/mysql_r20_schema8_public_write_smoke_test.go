package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

const smokePublicNotice = "v0.98-smoke-notice"

// TestR20Schema8MixedSmokeAfterPublicWrite applies public settings then runs the
// heterogeneous read bundle so post-apply smoke sees the written notice (R01).
func TestR20Schema8MixedSmokeAfterPublicWrite(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	current := settingsRead(t, f, "demo-admin", "public")
	settingsPut(t, f, "demo-admin", "public", current["revision"].(string), map[string]string{"notice": smokePublicNotice}, 200)
	seedGovernanceAudit(t, f, 8)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	loadPath := "/api/v1/messages/" + id + "/load?limit=10"
	governanceSettings := "/api/v1/cognitive/governance/settings"
	routes := []schema8ReadSmokeRoute{
		{path: "/ready", bearer: ""},
		{path: governanceSettings, bearer: "demo-admin", check: governancePublicNotice(smokePublicNotice)},
		{path: settingsBase + "public", bearer: "demo-admin", check: settingsPublicNotice(smokePublicNotice)},
		{path: governanceBase + "usage?limit=2&unit=CALL", bearer: "demo-admin", check: governanceItemsLen(2)},
		{path: loadPath, bearer: "demo-user", check: messageLoadLen(2)},
	}
	const workers = 30
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	start := time.Now()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		route := routes[i%len(routes)]
		go func(i int, route schema8ReadSmokeRoute) {
			defer wg.Done()
			base := f.host.URL
			if i%2 == 1 {
				base = second.URL
			}
			req, err := http.NewRequest("GET", base+route.path, nil)
			if err != nil {
				errs <- err
				return
			}
			if route.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+route.bearer)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errs <- err
				return
			}
			if resp.StatusCode != 200 {
				errs <- fmt.Errorf("%s status %d", route.path, resp.StatusCode)
				return
			}
			if route.check != nil {
				if err := route.check(body); err != nil {
					errs <- fmt.Errorf("%s: %w", route.path, err)
				}
			}
		}(i, route)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	t.Logf("mixed smoke after public write workers=%d elapsed=%s", workers, elapsed)
	if elapsed > 45*time.Second {
		t.Fatalf("smoke too slow %s", elapsed)
	}
}

func governancePublicNotice(want string) func([]byte) error {
	return func(body []byte) error {
		var page map[string]any
		if json.Unmarshal(body, &page) != nil {
			return fmt.Errorf("invalid json")
		}
		items, ok := page["items"].([]any)
		if !ok || len(items) != 1 {
			return fmt.Errorf("unexpected items")
		}
		row := items[0].(map[string]any)
		if row["value"] != want {
			return fmt.Errorf("notice %v want %s", row["value"], want)
		}
		return nil
	}
}

func settingsPublicNotice(want string) func([]byte) error {
	return func(body []byte) error {
		var doc map[string]any
		if json.Unmarshal(body, &doc) != nil {
			return fmt.Errorf("invalid json")
		}
		values, ok := doc["values"].(map[string]any)
		if !ok || values["notice"] != want {
			return fmt.Errorf("unexpected values %v", doc["values"])
		}
		return nil
	}
}