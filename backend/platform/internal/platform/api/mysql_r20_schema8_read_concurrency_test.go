package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
)

// TestR20Schema8ParallelGovernanceUsageReads hammers read-only governance usage paging
// across two HTTP hosts (light concurrency, not an SLO benchmark).
func TestR20Schema8ParallelGovernanceUsageReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	path := governanceBase + "usage?limit=2&unit=CALL"
	const workers = 32
	var wg sync.WaitGroup
	errs := make(chan error, workers)
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
			req.Header.Set("Authorization", "Bearer demo-admin")
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
			var page map[string]any
			if json.Unmarshal(body, &page) != nil {
				errs <- fmt.Errorf("invalid json")
				return
			}
			items, ok := page["items"].([]any)
			if !ok || len(items) != 2 {
				errs <- fmt.Errorf("unexpected page items %v", page["items"])
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// TestR20Schema8ParallelMessageLoadReads issues concurrent message history reads
// after a completed turn on schema8 relational storage.
func TestR20Schema8ParallelMessageLoadReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	path := "/api/v1/messages/" + id + "/load?limit=10"
	const workers = 24
	var wg sync.WaitGroup
	type snapshot struct {
		count int
		err   error
	}
	out := make(chan snapshot, workers)
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
				out <- snapshot{err: err}
				return
			}
			req.Header.Set("Authorization", "Bearer demo-user")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				out <- snapshot{err: err}
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				out <- snapshot{err: err}
				return
			}
			if resp.StatusCode != 200 {
				out <- snapshot{err: fmt.Errorf("status %d: %s", resp.StatusCode, body)}
				return
			}
			var page map[string]any
			if json.Unmarshal(body, &page) != nil {
				out <- snapshot{err: fmt.Errorf("invalid json")}
				return
			}
			data, ok := page["data"].([]any)
			if !ok {
				out <- snapshot{err: fmt.Errorf("missing data")}
				return
			}
			out <- snapshot{count: len(data)}
		}(i)
	}
	wg.Wait()
	close(out)
	want := 0
	for s := range out {
		if s.err != nil {
			t.Fatal(s.err)
		}
		if want == 0 {
			want = s.count
		} else if s.count != want {
			t.Fatal("inconsistent history length", want, s.count)
		}
	}
	if want != 2 {
		t.Fatal("expected user+assistant messages", want)
	}
}

// TestR20Schema8ParallelGovernanceExceptionsReads hits relational exception paging on dual hosts.
func TestR20Schema8ParallelGovernanceExceptionsReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	path := governanceBase + "exceptions?limit=1&status=OPEN"
	const workers = 32
	var wg sync.WaitGroup
	errs := make(chan error, workers)
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
			req.Header.Set("Authorization", "Bearer demo-admin")
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
			var page map[string]any
			if json.Unmarshal(body, &page) != nil {
				errs <- fmt.Errorf("invalid json")
				return
			}
			items, ok := page["items"].([]any)
			if !ok || len(items) != 1 {
				errs <- fmt.Errorf("unexpected exceptions page %v", page["items"])
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// TestR20Schema8ParallelGovernanceAuditReads hammers audit list queries after seeding events on schema8.
func TestR20Schema8ParallelGovernanceAuditReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	seedGovernanceAudit(t, f, 24)
	path := governanceBase + "audit?limit=5&query=audit-page-&outcome=success"
	const workers = 32
	var wg sync.WaitGroup
	errs := make(chan error, workers)
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
			req.Header.Set("Authorization", "Bearer demo-admin")
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
			var page map[string]any
			if json.Unmarshal(body, &page) != nil {
				errs <- fmt.Errorf("invalid json")
				return
			}
			items, ok := page["items"].([]any)
			if !ok || len(items) != 5 {
				errs <- fmt.Errorf("unexpected audit page %v", page["items"])
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}