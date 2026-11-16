package applications

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func exceptionPatch(status, revision, id string) *ExceptionPatch {
	return &ExceptionPatch{Status: status, Revision: revision, UpdatedAt: Now(), UpdatedBy: "demo-admin", Audit: []Event{{TenantID: "42", ID: id, OccurredAt: Now(), Action: "exception.update"}}}
}
func TestMySQLExceptionTargetedAtomicUpdateAndPreservation(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	called := false
	if _, handled, e := s.UpdateMySQLExceptionContext(context.Background(), "42", "target", func(map[string]any) (*ExceptionPatch, error) { called = true; return nil, nil }); handled || e != nil || called {
		t.Fatal("v1 did not fall back", e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',CAST(? AS JSON),'$.apps."channel-replenishment".future',CAST(? AS JSON))`, `[{"id":"target","tenant_id":"42","status":"OPEN","revision":"r1","future":{"n":9007199254740993}},{"id":"other","tenant_id":"42","status":"OPEN","revision":"old"}]`, `{"n":9007199254740993}`); e != nil {
		t.Fatal(e)
	}
	raw, handled, e := s.UpdateMySQLExceptionContext(context.Background(), "42", "target", func(item map[string]any) (*ExceptionPatch, error) {
		if item["revision"] != "r1" || item["status"] != "OPEN" {
			t.Fatal("wrong target")
		}
		return exceptionPatch("ACKNOWLEDGED", "r2", "first"), nil
	})
	if e != nil || !handled || !strings.Contains(string(raw), "9007199254740993") {
		t.Fatal("targeted response lost data", e, string(raw))
	}
	state := s.Read()
	if len(state.Audit) != 1 || state.Exceptions[0]["revision"] != "r2" || state.Exceptions[1]["revision"] != "old" {
		t.Fatal("wrong mutation scope")
	}
	read := func() string {
		var body string
		if e := db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&body); e != nil {
			t.Fatal(e)
		}
		return body
	}
	before := read()
	if strings.Count(before, "9007199254740993") != 2 {
		t.Fatal("opaque nested values changed")
	}
	if _, _, e = s.UpdateMySQLExceptionContext(context.Background(), "42", "target", func(map[string]any) (*ExceptionPatch, error) { return nil, errors.New("conflict") }); e == nil {
		t.Fatal("callback failure ignored")
	}
	if _, _, e = s.UpdateMySQLExceptionContext(context.Background(), "42", "target", func(map[string]any) (*ExceptionPatch, error) { return exceptionPatch("RESOLVED", "r3", "first"), nil }); e == nil {
		t.Fatal("duplicate audit committed")
	}
	if _, e = db.Exec(`ALTER TABLE cognara_fg_state ADD CONSTRAINT test_exception_status CHECK(JSON_UNQUOTE(JSON_EXTRACT(body,'$.exceptions[0].status'))<>'RESOLVED')`); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.UpdateMySQLExceptionContext(context.Background(), "42", "target", func(map[string]any) (*ExceptionPatch, error) {
		return exceptionPatch("RESOLVED", "r3", "rollback"), nil
	}); e == nil {
		t.Fatal("state constraint ignored")
	}
	if len(s.Read().Audit) != 1 || read() != before {
		t.Fatal("status and audit did not roll back together")
	}
	called = false
	if _, _, e = s.UpdateMySQLExceptionContext(context.Background(), "43", "target", func(map[string]any) (*ExceptionPatch, error) { called = true; return nil, nil }); !errors.Is(e, ErrExceptionUnavailable) || called {
		t.Fatal("cross-tenant target visible", e)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e = s.UpdateMySQLExceptionContext(canceled, "42", "target", func(map[string]any) (*ExceptionPatch, error) { called = true; return nil, nil }); e == nil || called {
		t.Fatal("canceled mutation ran")
	}
	if read() != before {
		t.Fatal("rejected operation changed state")
	}
}
func TestMySQLExceptionExactSelectionAndAliasFallback(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	raw := `[{"id":7,"tenant_id":"42","status":"OPEN","revision":"numeric"},{"id":"entry","tenant_id":"42 ","status":"OPEN","revision":"wrong-tenant"},{"id":"ENTRY","tenant_id":"42","status":"OPEN","revision":"uppercase"},{"id":"entry ","tenant_id":"42","status":"OPEN","revision":"space"},{"id":"entry","tenant_id":"42","status":"OPEN","revision":"first"},{"id":"entry","tenant_id":"42","status":"OPEN","revision":"second"},{"id":"7","tenant_id":"42","status":"OPEN","revision":"string"}]`
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',CAST(? AS JSON))`, raw); e != nil {
		t.Fatal(e)
	}
	if _, handled, e := s.UpdateMySQLExceptionContext(context.Background(), "42", "entry", func(item map[string]any) (*ExceptionPatch, error) {
		if item["revision"] != "first" {
			t.Fatal("selection changed Go semantics", item)
		}
		return exceptionPatch("ACKNOWLEDGED", "changed", "exact-1"), nil
	}); !handled || e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.UpdateMySQLExceptionContext(context.Background(), "42", "7", func(item map[string]any) (*ExceptionPatch, error) {
		if item["revision"] != "string" {
			t.Fatal("numeric id coerced to string")
		}
		return exceptionPatch("RESOLVED", "changed", "exact-2"), nil
	}); e != nil {
		t.Fatal(e)
	}
	state := s.Read()
	if state.Exceptions[4]["status"] != "ACKNOWLEDGED" || state.Exceptions[5]["status"] != "OPEN" || state.Exceptions[0]["revision"] != "numeric" {
		t.Fatal("updated duplicate or coerced id")
	}
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.Exceptions',JSON_ARRAY())`); e != nil {
		t.Fatal(e)
	}
	called := false
	if _, handled, e := s.UpdateMySQLExceptionContext(context.Background(), "42", "entry", func(map[string]any) (*ExceptionPatch, error) { called = true; return nil, nil }); handled || e != nil || called {
		t.Fatal("alias did not fall back", e)
	}
}
