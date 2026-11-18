package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// schema8ParallelAuthorizedGET runs concurrent GETs across two HTTP hosts and returns wall time.
func schema8ParallelAuthorizedGET(t *testing.T, f *fixture, second *httptest.Server, path, bearer string, workers int, validate func([]byte) error) time.Duration {
	t.Helper()
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	start := time.Now()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			base := f.host.URL
			if i%2 == 1 {
				base = second.URL
			}
			req, err := http.NewRequest("GET", base+path, nil)
			if err != nil {
				errs <- err
				return
			}
			req.Header.Set("Authorization", "Bearer "+bearer)
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
				errs <- fmt.Errorf("status %d: %s", resp.StatusCode, body)
				return
			}
			if validate != nil {
				if err := validate(body); err != nil {
					errs <- err
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	return time.Since(start)
}

// TestR20Schema8TieredGovernanceUsageLoad steps read-only usage paging workers (32 → 64)
// and records elapsed time per tier (light probe, not an SLO benchmark).
func TestR20Schema8TieredGovernanceUsageLoad(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	path := governanceBase + "usage?limit=2&unit=CALL"
	validate := func(body []byte) error {
		var page map[string]any
		if json.Unmarshal(body, &page) != nil {
			return fmt.Errorf("invalid json")
		}
		items, ok := page["items"].([]any)
		if !ok || len(items) != 2 {
			return fmt.Errorf("unexpected page items %v", page["items"])
		}
		return nil
	}
	for _, tier := range []struct {
		workers int
		limit   time.Duration
	}{
		{32, 12 * time.Second},
		{64, 25 * time.Second},
	} {
		elapsed := schema8ParallelAuthorizedGET(t, f, second, path, "demo-admin", tier.workers, validate)
		t.Logf("governance usage tier workers=%d elapsed=%s", tier.workers, elapsed)
		if elapsed > tier.limit {
			t.Fatalf("tier workers=%d exceeded wall limit %s (elapsed %s)", tier.workers, tier.limit, elapsed)
		}
	}
}

// TestR20Schema8TieredMessageLoadLoad steps message history reads after one completed turn.
func TestR20Schema8TieredMessageLoadLoad(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	path := "/api/v1/messages/" + id + "/load?limit=10"
	var wantLen atomic.Int32
	validate := func(body []byte) error {
		var page map[string]any
		if json.Unmarshal(body, &page) != nil {
			return fmt.Errorf("invalid json")
		}
		data, ok := page["data"].([]any)
		if !ok {
			return fmt.Errorf("missing data")
		}
		n := int32(len(data))
		prev := wantLen.Load()
		if prev == 0 {
			wantLen.CompareAndSwap(0, n)
			prev = wantLen.Load()
		}
		if prev != n {
			return fmt.Errorf("inconsistent history length want=%d got=%d", prev, n)
		}
		return nil
	}
	for _, tier := range []struct {
		workers int
		limit   time.Duration
	}{
		{32, 15 * time.Second},
		{64, 30 * time.Second},
		{96, 45 * time.Second},
	} {
		elapsed := schema8ParallelAuthorizedGET(t, f, second, path, "demo-user", tier.workers, validate)
		t.Logf("message load tier workers=%d elapsed=%s", tier.workers, elapsed)
		if elapsed > tier.limit {
			t.Fatalf("tier workers=%d exceeded wall limit %s (elapsed %s)", tier.workers, tier.limit, elapsed)
		}
	}
	if got := int(wantLen.Load()); got != 2 {
		t.Fatal("expected user+assistant messages", got)
	}
}