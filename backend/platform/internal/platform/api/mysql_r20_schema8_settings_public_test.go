package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
)

// TestR20Schema8ParallelGovernancePublicSettingsReads hammers the enterprise public
// notice list (R01) across two HTTP hosts on schema8 relational storage.
func TestR20Schema8ParallelGovernancePublicSettingsReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	wantNotice := "v0.96 并发探针公告"
	current := settingsRead(t, f, "demo-admin", "public")
	settingsPut(t, f, "demo-admin", "public", current["revision"].(string), map[string]string{"notice": wantNotice}, 200)
	path := "/api/v1/cognitive/governance/settings"
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
				errs <- fmt.Errorf("unexpected items %v", page["items"])
				return
			}
			row := items[0].(map[string]any)
			if row["value"] != wantNotice {
				errs <- fmt.Errorf("notice mismatch %v", row["value"])
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// TestR20Schema8ParallelSettingsPublicSectionReads issues concurrent GET of the
// settings/public section document on both hosts (admin read path).
func TestR20Schema8ParallelSettingsPublicSectionReads(t *testing.T) {
	f, _, second := schema8DualHTTPFixture(t)
	wantNotice := "v0.96 settings public section"
	current := settingsRead(t, f, "demo-admin", "public")
	settingsPut(t, f, "demo-admin", "public", current["revision"].(string), map[string]string{"notice": wantNotice}, 200)
	path := settingsBase + "public"
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
			var doc map[string]any
			if json.Unmarshal(body, &doc) != nil {
				errs <- fmt.Errorf("invalid json")
				return
			}
			values, ok := doc["values"].(map[string]any)
			if !ok || values["notice"] != wantNotice {
				errs <- fmt.Errorf("unexpected values %v", doc["values"])
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}