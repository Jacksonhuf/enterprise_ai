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

const smokeWriteReadNotice = "v0.99-write-read-notice"
const smokeWriteReadProfile = "v0.99-write-read-profile"

// TestR20Schema8MixedSmokeWriteThenReadOrchestration runs a write phase (public + profile)
// on schema8, then concurrent reads on both hosts (post-apply smoke with write stage).
func TestR20Schema8MixedSmokeWriteThenReadOrchestration(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	writeStart := time.Now()
	pub := settingsRead(t, f, "demo-admin", "public")
	settingsPut(t, f, "demo-admin", "public", pub["revision"].(string), map[string]string{"notice": smokeWriteReadNotice}, 200)
	prof := settingsRead(t, f, "demo-user", "profile")
	settingsPut(t, f, "demo-user", "profile", prof["revision"].(string), map[string]string{"display_name": smokeWriteReadProfile}, 200)
	writeElapsed := time.Since(writeStart)
	t.Logf("write phase public+profile elapsed=%s", writeElapsed)

	seedGovernanceAudit(t, f, 6)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	loadPath := "/api/v1/messages/" + id + "/load?limit=10"
	routes := []schema8ReadSmokeRoute{
		{path: "/ready", bearer: ""},
		{path: "/api/v1/cognitive/governance/settings", bearer: "demo-admin", check: governancePublicNotice(smokeWriteReadNotice)},
		{path: settingsBase + "public", bearer: "demo-admin", check: settingsPublicNotice(smokeWriteReadNotice)},
		{path: settingsBase + "profile", bearer: "demo-user", check: settingsProfileDisplayName(smokeWriteReadProfile)},
		{path: governanceBase + "grants?limit=2&status=ACTIVE", bearer: "demo-admin", check: governanceItemsLen(2)},
		{path: loadPath, bearer: "demo-user", check: messageLoadLen(2)},
	}
	const workers = 36
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	readStart := time.Now()
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
	readElapsed := time.Since(readStart)
	t.Logf("read phase workers=%d elapsed=%s", workers, readElapsed)
	if writeElapsed > 20*time.Second || readElapsed > 45*time.Second {
		t.Fatalf("smoke write=%s read=%s", writeElapsed, readElapsed)
	}
}

func settingsProfileDisplayName(want string) func([]byte) error {
	return func(body []byte) error {
		var doc map[string]any
		if json.Unmarshal(body, &doc) != nil {
			return fmt.Errorf("invalid json")
		}
		values, ok := doc["values"].(map[string]any)
		if !ok || values["display_name"] != want {
			return fmt.Errorf("unexpected profile %v", doc["values"])
		}
		return nil
	}
}