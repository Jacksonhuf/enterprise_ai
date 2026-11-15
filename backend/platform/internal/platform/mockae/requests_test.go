package mockae

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanonicalRecoveryDigestAndStrictRequests(t *testing.T) {
	decode := func(raw string) (string, error) {
		var value struct {
			ID       string `json:"id"`
			Optional int    `json:"optional"`
		}
		return decodeRequest(httptest.NewRequest("POST", "/", strings.NewReader(raw)), &value)
	}
	a, e := decode(`{"id":"sample","optional":1}`)
	if e != nil {
		t.Fatal(e)
	}
	b, e := decode(" { \"optional\" : 1, \"id\" : \"sample\" } ")
	if e != nil || a != b || !validDigest(a) {
		t.Fatal("equivalent JSON did not preserve recovery digest")
	}
	c, e := decode(`{"id":"sample"}`)
	if e != nil || a == c {
		t.Fatal("omitted optional field was normalized away")
	}
	if _, e = decode(`{"id":"sample","tenant_id":"forged"}`); e == nil {
		t.Fatal("unknown identity field accepted")
	}
	for _, bad := range []string{"sha256:bad", strings.ToUpper(a), ""} {
		if validDigest(bad) {
			t.Fatal("invalid digest accepted")
		}
	}
}
