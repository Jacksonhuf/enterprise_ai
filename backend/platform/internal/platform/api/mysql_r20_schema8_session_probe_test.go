package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
)

func postSessionHTTP(t *testing.T, base, identity, path, title string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"title": title})
	req, err := http.NewRequest("POST", base+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+identity)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("POST %s: %d %s", base+path, resp.StatusCode, raw)
	}
	out := decode(t, raw)
	id, _ := out["session_id"].(string)
	if id == "" {
		t.Fatal("session_id", out)
	}
	return id
}

// TestR20Schema8DualHostSessionCreateOnSecondHost ensures session create succeeds on the peer API host.
func TestR20Schema8DualHostSessionCreateOnSecondHost(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	path := "/api/v1/cognitive/apps/channel-replenishment/sessions"
	id := postSessionHTTP(t, second.URL, "demo-user", path, "schema8-host-b")
	if id == "" {
		t.Fatal("empty session")
	}
	if _, ok := f.store.Read().Sessions[id]; !ok {
		t.Fatal("session not visible on primary store")
	}
}

// TestR20Schema8DualHostConcurrentSessionListReads hammers session catalog reads on both hosts.
func TestR20Schema8DualHostConcurrentSessionListReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	pathCreate := "/api/v1/cognitive/apps/channel-replenishment/sessions"
	_ = postSessionHTTP(t, f.host.URL, "demo-user", pathCreate, "list-probe")
	listPath := "/api/v1/sessions?app_id=channel-replenishment&limit=20"
	const workers = 24
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
			req, err := http.NewRequest("GET", base+listPath, nil)
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
			if !ok || len(items) < 1 {
				errs <- fmt.Errorf("short page %v", page["items"])
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// TestR20Schema8DualHostCrossHostChatProbe creates a session on host A, runs agent-chat
// on host B, and reads message history from both hosts.
func TestR20Schema8DualHostCrossHostChatProbe(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	path := "/api/v1/cognitive/apps/channel-replenishment/sessions"
	sessID := postSessionHTTP(t, f.host.URL, "demo-user", path, "cross-host-probe")
	chatBody, _ := json.Marshal(map[string]string{"query": "SO1002需求增加到350"})
	req, _ := http.NewRequest("POST", second.URL+"/api/v1/agent-chat/"+sessID, bytes.NewReader(chatBody))
	req.Header.Set("Authorization", "Bearer demo-user")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("agent-chat", resp.StatusCode)
	}
	loadPath := "/api/v1/messages/" + sessID + "/load?limit=10"
	var wg sync.WaitGroup
	type snapshot struct {
		n   int
		err error
	}
	out := make(chan snapshot, 2)
	for _, base := range []string{f.host.URL, second.URL} {
		wg.Add(1)
		go func(base string) {
			defer wg.Done()
			req, err := http.NewRequest("GET", base+loadPath, nil)
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
			out <- snapshot{n: len(data)}
		}(base)
	}
	wg.Wait()
	close(out)
	want := 0
	for s := range out {
		if s.err != nil {
			t.Fatal(s.err)
		}
		if want == 0 {
			want = s.n
		} else if s.n != want {
			t.Fatal("history mismatch", want, s.n)
		}
	}
	if want < 2 {
		t.Fatal("expected user+assistant messages", want)
	}
}