package mockae

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHistoricalGapFixedPoint(t *testing.T) {
	v, e := Calculate(350)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(v)
	for _, q := range []string{"336.6", "276.6", "333.3", "13.3", "70", "110"} {
		if !strings.Contains(string(b), q) {
			t.Fatalf("missing %s: %s", q, b)
		}
	}
	for _, q := range []int{-1, 0, 100001} {
		if _, e = Calculate(q); e == nil {
			t.Fatalf("accepted invalid %d", q)
		}
	}
	v, e = Calculate(10)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(v["summary"].(string), "待产0") {
		t.Fatal(v)
	}
}
