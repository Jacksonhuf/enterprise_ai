package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAssetDiscoveryRestrictsReleaseAndContextHidesReceipt(t *testing.T) {
	var calls atomic.Int32
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/internal/v1/assets/discover":
			fmt.Fprint(w, `{"items":[{"asset_id":"inventory-policy","name":"库存制度"},{"asset_id":"chip-gap-metric","name":"未引用指标"}],"next_cursor":null}`)
		case "/internal/v1/assets/context":
			fmt.Fprint(w, `{"items":[{"asset_id":"inventory-policy","resolved_version":"v1","professional_content":{"definition":"先核对库存口径"},"execution_readiness":"NOT_APPLICABLE","usage_constraints":[],"related_refs":[]}],"context_receipt":"do-not-expose-this-receipt"}`)
		default:
			t.Error("unexpected request")
		}
	}))
	defer host.Close()
	request := runtimeRequest(t)
	request.Release.Config.CapabilityRefs.AssetRefs = []app.AssetRef{{AssetID: "inventory-policy"}}
	run := &serviceTool{client: &cognitive.Client{BaseURL: host.URL}, request: request, authorize: func(governance.Principal, string, string, string) bool { return true }, resources: map[string]ResourceCheck{}}
	discover := &assetTool{kind: "cognitive_discover_assets", run: run}
	out, e := discover.Execute(context.Background(), json.RawMessage(`{"query":""}`))
	if e != nil || strings.Contains(out.Output, "chip-gap-metric") || !strings.Contains(out.Output, "inventory-policy") {
		t.Fatalf("discovery leaked an unselected asset: %v %+v", e, out)
	}
	tool := &assetTool{kind: "cognitive_get_asset_context", run: run}
	out, e = tool.Execute(context.Background(), json.RawMessage(`{"asset_id":"inventory-policy"}`))
	if e != nil || strings.Contains(out.Output, "context_receipt") || strings.Contains(out.Output, "do-not-expose") {
		t.Fatal("context credential exposed", e)
	}
	count := calls.Load()
	for _, args := range []string{`{"asset_id":"chip-gap-metric"}`, `{"asset_id":"inventory-policy","tenant_id":"injected"}`, `{"asset_id":"inventory-policy","version":"injected"}`} {
		if _, e = tool.Execute(context.Background(), json.RawMessage(args)); e == nil {
			t.Fatal("forbidden input accepted")
		}
	}
	if calls.Load() != count {
		t.Fatal("forbidden input reached backend")
	}
}
func TestContextRevocationBeforeOutputIsDenied(t *testing.T) {
	var revoked atomic.Bool
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		revoked.Store(true)
		fmt.Fprint(w, `{"items":[{"asset_id":"inventory-policy","resolved_version":"v1"}],"context_receipt":"opaque"}`)
	}))
	defer host.Close()
	run := &serviceTool{client: &cognitive.Client{BaseURL: host.URL}, request: runtimeRequest(t), authorize: func(governance.Principal, string, string, string) bool { return !revoked.Load() }, resources: map[string]ResourceCheck{}}
	tool := &assetTool{kind: "cognitive_get_asset_context", run: run}
	if _, e := tool.Execute(context.Background(), json.RawMessage(`{"asset_id":"inventory-policy"}`)); e == nil {
		t.Fatal("context leaked after revocation")
	}
}
func TestNotReadyMetricNeverExecutes(t *testing.T) {
	var executions atomic.Int32
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/v1/executions" {
			executions.Add(1)
		}
		fmt.Fprint(w, `{"items":[{"asset_id":"chip-gap-metric","resolved_version":"v1","execution_readiness":"NOT_READY"}],"context_receipt":"opaque"}`)
	}))
	defer host.Close()
	run := &serviceTool{client: &cognitive.Client{BaseURL: host.URL}, request: runtimeRequest(t), authorize: func(governance.Principal, string, string, string) bool { return true }, resources: map[string]ResourceCheck{}}
	tool := &assetTool{kind: "cognitive_execute", run: run}
	if _, e := tool.Execute(context.Background(), json.RawMessage(`{"asset_id":"chip-gap-metric","parameters":{"order_id":"SO1002","target_quantity":350}}`)); e == nil || executions.Load() != 0 {
		t.Fatal("not-ready context executed")
	}
}
