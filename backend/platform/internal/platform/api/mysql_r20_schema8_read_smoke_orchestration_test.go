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

type schema8ReadSmokeRoute struct {
	path   string
	bearer string
	check  func([]byte) error
}

// TestR20Schema8MixedReadOnlySmokeOrchestration exercises heterogeneous read paths
// on both HTTP hosts in one probe (post-apply smoke bundle, not an SLO benchmark).
func TestR20Schema8MixedReadOnlySmokeOrchestration(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	seedGovernanceAudit(t, f, 12)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	loadPath := "/api/v1/messages/" + id + "/load?limit=10"
	routes := []schema8ReadSmokeRoute{
		{path: "/ready", bearer: "", check: func(body []byte) error {
			if len(body) == 0 {
				return fmt.Errorf("empty ready body")
			}
			return nil
		}},
		{path: governanceBase + "usage?limit=2&unit=CALL", bearer: "demo-admin", check: governanceItemsLen(2)},
		{path: governanceBase + "grants?limit=2&status=ACTIVE", bearer: "demo-admin", check: governanceItemsLen(2)},
		{path: governanceBase + "exceptions?limit=1&status=OPEN", bearer: "demo-admin", check: governanceItemsLen(1)},
		{path: governanceBase + "audit?limit=5&query=audit-page-&outcome=success", bearer: "demo-admin", check: governanceItemsLen(5)},
		{path: loadPath, bearer: "demo-user", check: messageLoadLen(2)},
	}
	const workers = 36
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
				errs <- fmt.Errorf("%s status %d: %s", route.path, resp.StatusCode, body)
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
	t.Logf("mixed read-only smoke routes=%d workers=%d elapsed=%s", len(routes), workers, elapsed)
	if elapsed > 45*time.Second {
		t.Fatalf("mixed smoke exceeded wall limit (elapsed %s)", elapsed)
	}
}

func governanceItemsLen(want int) func([]byte) error {
	return func(body []byte) error {
		var page map[string]any
		if json.Unmarshal(body, &page) != nil {
			return fmt.Errorf("invalid json")
		}
		items, ok := page["items"].([]any)
		if !ok || len(items) != want {
			return fmt.Errorf("unexpected items len %v want %d", page["items"], want)
		}
		return nil
	}
}

func messageLoadLen(want int) func([]byte) error {
	return func(body []byte) error {
		var page map[string]any
		if json.Unmarshal(body, &page) != nil {
			return fmt.Errorf("invalid json")
		}
		data, ok := page["data"].([]any)
		if !ok || len(data) != want {
			return fmt.Errorf("unexpected data len %v want %d", page["data"], want)
		}
		return nil
	}
}