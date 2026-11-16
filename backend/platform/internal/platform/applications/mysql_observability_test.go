package applications

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func failureRecord(id string) map[string]any {
	return map[string]any{"id": id, "tenant_id": "42", "revision": "r-" + id, "status": "OPEN", "name": "服务请求未完成", "request_id": "request-" + id, "code": "CAPABILITY_UNAVAILABLE", "http_status": 503, "occurred_at": Now(), "updated_at": Now()}
}
func TestMySQLFailureAuditAndExceptionAtomicAppend(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	event := Event{TenantID: "42", ID: "first", OccurredAt: Now(), RequestID: "request-first", Outcome: "failed"}
	if handled, e := s.AppendMySQLObservabilityContext(context.Background(), []Event{event}, failureRecord("first")); handled || e != nil {
		t.Fatal("v1 did not fall back", e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',CAST(? AS JSON),'$.apps."channel-replenishment".future',CAST(? AS JSON))`, `[{"id":"old","tenant_id":"42","status":"OPEN","future":{"keep":9007199254740993}}]`, `{"keep":9007199254740993}`); e != nil {
		t.Fatal(e)
	}
	if handled, e := s.AppendMySQLObservabilityContext(context.Background(), []Event{event}, failureRecord("first")); !handled || e != nil {
		t.Fatal("failure append", e)
	}
	st := s.Read()
	if len(st.Audit) != 1 || len(st.Exceptions) != 2 || st.Exceptions[1]["request_id"] != st.Audit[0].RequestID {
		t.Fatal("unbound failure record")
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
		t.Fatal("failure rewrote unknown nested values")
	}
	if _, e = s.AppendMySQLObservabilityContext(context.Background(), []Event{{TenantID: "42", ID: "rolled-back", OccurredAt: Now()}, event}, failureRecord("duplicate")); e == nil {
		t.Fatal("duplicate audit accepted")
	}
	if len(s.Read().Audit) != 1 || read() != before {
		t.Fatal("partial audit batch committed")
	}
	// Force the state UPDATE to fail AFTER the audit insert, proving cross-table rollback.
	if _, e = db.Exec(`ALTER TABLE cognara_fg_state ADD CONSTRAINT test_exception_limit CHECK(JSON_LENGTH(JSON_EXTRACT(body,'$.exceptions'))<=2)`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AppendMySQLObservabilityContext(context.Background(), []Event{{TenantID: "42", ID: "body-rollback", OccurredAt: Now()}}, failureRecord("third")); e == nil {
		t.Fatal("state constraint ignored")
	}
	if len(s.Read().Audit) != 1 || read() != before {
		t.Fatal("audit committed without exception")
	}
	if _, e = db.Exec(`ALTER TABLE cognara_fg_state DROP CHECK test_exception_limit`); e != nil {
		t.Fatal(e)
	}
	validationEvent := Event{TenantID: "42", ID: "validation-new", OccurredAt: Now()}
	invalid := failureRecord("invalid")
	invalid["tenant_id"] = "43"
	if _, e = s.AppendMySQLObservabilityContext(context.Background(), []Event{validationEvent}, invalid); e == nil {
		t.Fatal("tenant mismatch accepted")
	}
	if _, e = s.AppendMySQLObservabilityContext(context.Background(), nil, failureRecord("no-audit")); e == nil {
		t.Fatal("unaudited exception accepted")
	}
	invalid = failureRecord("bad-json")
	invalid["future"] = make(chan int)
	if _, e = s.AppendMySQLObservabilityContext(context.Background(), []Event{validationEvent}, invalid); e == nil {
		t.Fatal("invalid exception JSON accepted")
	}
	if read() != before {
		t.Fatal("invalid exception modified body")
	}
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.Exceptions',JSON_ARRAY())`); e != nil {
		t.Fatal(e)
	}
	if handled, e := s.AppendMySQLObservabilityContext(context.Background(), []Event{event}, failureRecord("alias")); handled || e != nil {
		t.Fatal("exception alias did not fall back", e)
	}
}
func TestMySQLFailureAppendConcurrentStoresAndNullContainer(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	a, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',NULL,'$.future',CAST(? AS JSON))`, `{"keep":true}`); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i, store := range []*Store{a, b} {
		wg.Add(1)
		go func(i int, s *Store) {
			defer wg.Done()
			id := fmt.Sprint(i)
			_, e := s.AppendMySQLObservabilityContext(context.Background(), []Event{{TenantID: "42", ID: id, OccurredAt: Now(), RequestID: "request-" + id}}, failureRecord(id))
			errs <- e
		}(i, store)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	state := a.Read()
	if len(state.Audit) != 2 || len(state.Exceptions) != 2 {
		t.Fatal("concurrent failure lost")
	}
	for _, event := range state.Audit {
		found := false
		for _, item := range state.Exceptions {
			if item["request_id"] == event.RequestID {
				found = true
			}
		}
		if !found {
			t.Fatal("failure association lost")
		}
	}
	var value string
	if e = db.QueryRow(`SELECT JSON_EXTRACT(body,'$.future') FROM cognara_fg_state WHERE id=1`).Scan(&value); e != nil || !strings.Contains(value, "true") {
		t.Fatal("unrelated domain changed", e)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = a.AppendMySQLObservabilityContext(canceled, []Event{{TenantID: "42", ID: "canceled", OccurredAt: Now()}}, failureRecord("canceled")); e == nil {
		t.Fatal("canceled transaction committed")
	}
	if len(a.Read().Exceptions) != 2 || len(a.Read().Audit) != 2 {
		t.Fatal("cancellation mutated data")
	}
}
