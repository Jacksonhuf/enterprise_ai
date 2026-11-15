package runtime

import (
	"context"
	"github.com/Tencent/WeKnora/internal/platform/connectors/cognitive"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"testing"
)

func TestCanceledTurnDoesNotDispatchBusinessCall(t *testing.T) {
	models, _ := fixtureModel(t, `{"service_id":"replenishment-gap","arguments":{"order_id":"SO1002","target_quantity":350}}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine := &Engine{Models: models, Client: &cognitive.Client{BaseURL: "http://127.0.0.1:1"}, Authorize: func(governance.Principal, string, string, string) bool {
		t.Error("canceled turn attempted a tool")
		return false
	}}
	if _, e := engine.Run(ctx, runtimeRequest(t)); e == nil {
		t.Fatal("canceled turn succeeded")
	}
}
