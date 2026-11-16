package applications

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLExceptionQueryReadOnlySnapshotAndFallback(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	match := func(map[string]any) bool { return true }
	idOf := func(row map[string]any) string { id, _ := row["id"].(string); return id }
	if _, _, handled, e := s.QueryMySQLExceptions(context.Background(), "42", "", 20, match, idOf); handled || e != nil {
		t.Fatal("v1 did not fall back", e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	db := preflightDB(t, dsn)
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions',CAST(? AS JSON))`, `[{"id":"local","tenant_id":"42","future":{"n":9007199254740993}},{"id":"foreign","tenant_id":"43"},{"id":"numeric","tenant_id":42},{"id":"space","tenant_id":"42 "}]`); e != nil {
		t.Fatal(e)
	}
	var before, after string
	if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&before); e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.exceptions[0].id','uncommitted') WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	type result struct {
		rows          []json.RawMessage
		more, handled bool
		err           error
	}
	done := make(chan result, 1)
	go func() {
		rows, more, handled, e := s.QueryMySQLExceptions(ctx, "42", "", 20, match, idOf)
		done <- result{rows, more, handled, e}
	}()
	select {
	case out := <-done:
		if out.err != nil || !out.handled || out.more || len(out.rows) != 1 || !strings.Contains(string(out.rows[0]), "local") || !strings.Contains(string(out.rows[0]), "9007199254740993") {
			t.Fatal("inconsistent or cross-tenant projection", out.err, out.rows)
		}
	case <-ctx.Done():
		t.Fatal("query waited for writer or local mutex")
	}
	if e = tx.Rollback(); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&after); e != nil || before != after {
		t.Fatal("query changed body", e)
	}
	if _, _, _, e = s.QueryMySQLExceptions(context.Background(), "42", "missing", 20, match, idOf); e != ErrExceptionPageCursor {
		t.Fatal("invalid anchor accepted", e)
	}
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.Exceptions',JSON_ARRAY())`); e != nil {
		t.Fatal(e)
	}
	if _, _, handled, e := s.QueryMySQLExceptions(context.Background(), "42", "", 20, match, idOf); handled || e != nil {
		t.Fatal("alias did not fall back", e)
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if _, _, handled, e := s.QueryMySQLExceptions(canceled, "42", "", 20, match, idOf); !handled || e == nil {
		t.Fatal("canceled query reported success")
	}
}
