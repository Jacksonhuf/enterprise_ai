package governance

import (
	"strings"
	"testing"
)

func TestDelegationRejectsTamperingAndWrongKey(t *testing.T) {
	p, _ := DemoIdentity("demo-user")
	token := Sign(p, []byte("test-key"))
	got, e := Verify(token, []byte("test-key"))
	if e != nil || got.UserID != p.UserID {
		t.Fatal(e)
	}
	if _, e = Verify(token, []byte("wrong-key")); e == nil {
		t.Fatal("wrong key accepted")
	}
	parts := strings.Split(token, ".")
	parts[0] = "e30"
	if _, e = Verify(strings.Join(parts, "."), []byte("test-key")); e == nil {
		t.Fatal("modified token accepted")
	}
	p.Audience = "other"
	if _, e = Verify(Sign(p, []byte("test-key")), []byte("test-key")); e == nil {
		t.Fatal("wrong audience accepted")
	}
	if _, e = DemoIdentity("production-looking-token"); e == nil {
		t.Fatal("unknown identity accepted")
	}
}
