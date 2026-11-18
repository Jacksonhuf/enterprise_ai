package api_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	api "github.com/Tencent/WeKnora/internal/platform/api"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
)

func schema8DualHTTPFixture(t *testing.T) (*fixture, string, *httptest.Server) {
	t.Helper()
	f, dsn, _ := mysqlSchema8Fixture(t)
	b, err := app.OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.Close() })
	second := &api.Server{Store: b, Client: f.client, ServiceToken: f.server.ServiceToken, SigningKey: f.server.SigningKey}
	host := httptest.NewServer(second.Handler())
	t.Cleanup(host.Close)
	return f, dsn, host
}

func getBytes(t *testing.T, identity, url string) []byte {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+identity)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s: status %d: %s", url, resp.StatusCode, body)
	}
	return body
}

// TestR20Schema8DualHostContinueStreamReplay ensures completed-turn SSE replay is
// served from durable state on either API host without re-running the business turn.
func TestR20Schema8DualHostContinueStreamReplay(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	answer := f.store.Read().Sessions[id].Messages[1]
	usageBefore := len(f.store.Read().Usage)
	msgsBefore := len(f.store.Read().Sessions[id].Messages)
	path := "/api/v1/sessions/continue-stream/" + id + "?message_id=" + answer.ID
	for _, base := range []string{f.host.URL, second.URL} {
		out := getBytes(t, "demo-user", base+path)
		if !bytes.Contains(out, []byte("276.6")) || !bytes.Contains(out, []byte("event: done")) {
			t.Fatal("continue-stream replay missing payload", base, out)
		}
	}
	if len(f.store.Read().Usage) != usageBefore || len(f.store.Read().Sessions[id].Messages) != msgsBefore {
		t.Fatal("SSE replay re-ran business turn")
	}
}

// TestR20Schema8DualHostContinueStreamMissingMessage ensures both hosts agree on 404
// for unknown assistant message ids (relational session read).
func TestR20Schema8DualHostContinueStreamMissingMessage(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	id := f.session()
	path := "/api/v1/sessions/continue-stream/" + id + "?message_id=missing-assistant"
	for _, base := range []string{f.host.URL, second.URL} {
		req, _ := http.NewRequest("GET", base+path, nil)
		req.Header.Set("Authorization", "Bearer demo-user")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Fatal("unknown message must 404", base, resp.StatusCode)
		}
	}
}

// TestR20Schema8ParallelContinueStreamReplay hammers completed-turn SSE replay on both hosts
// without re-executing the business turn (light concurrency, not an SLO benchmark).
func TestR20Schema8ParallelContinueStreamReplay(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	id := f.session()
	f.request("demo-user", "POST", "/api/v1/agent-chat/"+id, map[string]string{"query": "SO1002需求增加到350"}, 200)
	answer := f.store.Read().Sessions[id].Messages[1]
	usageBefore := len(f.store.Read().Usage)
	msgsBefore := len(f.store.Read().Sessions[id].Messages)
	path := "/api/v1/sessions/continue-stream/" + id + "?message_id=" + answer.ID
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
			req.Header.Set("Authorization", "Bearer demo-user")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errs <- err
				return
			}
			out, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != 200 {
				errs <- fmt.Errorf("status %d: %v", resp.StatusCode, err)
				return
			}
			if !bytes.Contains(out, []byte("276.6")) || !bytes.Contains(out, []byte("event: done")) {
				errs <- fmt.Errorf("replay missing payload from %s", base)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if len(f.store.Read().Usage) != usageBefore || len(f.store.Read().Sessions[id].Messages) != msgsBefore {
		t.Fatal("parallel replay re-ran business turn")
	}
}

// TestR20Schema8ParallelReadyAcrossDualHosts is a light concurrency probe (not an SLO bench).
func TestR20Schema8ParallelReadyAcrossDualHosts(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	const workers = 48
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
			resp, err := http.Get(base + "/ready")
			if err != nil {
				errs <- err
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				errs <- fmt.Errorf("ready status %d", resp.StatusCode)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 15*time.Second {
		t.Fatal("ready probe too slow", d)
	}
}