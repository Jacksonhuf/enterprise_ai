package api_test

import (
	"reflect"
	"testing"
)

func TestServiceContractInspectionPreservesPublishedMetadataWithoutExecution(t *testing.T) {
	f := setup(t)
	before := f.store.Read()
	for _, identity := range []string{"demo-engineer", "demo-admin"} {
		listing := decode(t, f.request(identity, "GET", "/api/v1/cognitive/selectors/services?query=补货&limit=1", nil, 200))
		items := listing["items"].([]any)
		if len(items) != 1 {
			t.Fatal(listing)
		}
		service := items[0].(map[string]any)
		if service["service_id"] != "replenishment-gap" || service["service_version"] != "v1.1" || service["contract_revision"] != "gap-contract-v1" {
			t.Fatal(service)
		}
		input := service["input_schema"].(map[string]any)
		properties := input["properties"].(map[string]any)
		quantity := properties["target_quantity"].(map[string]any)
		if quantity["type"] != "integer" || quantity["minimum"] != float64(1) || quantity["maximum"] != float64(100000) {
			t.Fatal(quantity)
		}
		if !reflect.DeepEqual(input["required"], []any{"order_id", "target_quantity"}) || input["additionalProperties"] != false {
			t.Fatal(input)
		}
		output := service["output_schema"].(map[string]any)
		if len(output) != 1 || output["type"] != "object" {
			t.Fatal("do not fabricate unprovided output fields", output)
		}
	}
	f.request("demo-user", "GET", "/api/v1/cognitive/selectors/services", nil, 403)
	f.request("demo-outsider", "GET", "/api/v1/cognitive/selectors/services", nil, 403)
	after := f.store.Read()
	if !reflect.DeepEqual(before.Usage, after.Usage) || !reflect.DeepEqual(before.Receipts, after.Receipts) || !reflect.DeepEqual(before.Sessions, after.Sessions) {
		t.Fatal("inspection must not execute or create a session")
	}
}
