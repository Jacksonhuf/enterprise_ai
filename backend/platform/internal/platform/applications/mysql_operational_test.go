package applications

import (
	"context"
	"fmt"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"strconv"
	"sync"
	"testing"
)

func TestMySQLOperationalAuditBufferAndAtomicRollback(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	s, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Update(func(st *State) error {
		for i := 0; i < 310; i++ {
			st.Audit = append(st.Audit, Event{ID: fmt.Sprintf("old-%d", i), TenantID: "42", OccurredAt: Now()})
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	if len(s.ReadOperational().Audit) != 0 || len(s.Read().Audit) != 310 {
		t.Fatal("operational read exposed history")
	}
	if e = s.UpdateOperational(func(st *State) error {
		if len(st.Audit) != 0 {
			t.Fatal("callback received history")
		}
		st.Settings["new"] = []byte(`"kept"`)
		st.Audit = append(st.Audit, Event{ID: "new", TenantID: "42", OccurredAt: Now()})
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = s.UpdateOperational(func(st *State) error {
		st.Settings["new"] = []byte(`"lost"`)
		st.Audit = append(st.Audit, Event{ID: "old-0", TenantID: "42", OccurredAt: Now()})
		return nil
	}); e == nil {
		t.Fatal("duplicate event committed")
	}
	state := s.Read()
	if len(state.Audit) != 311 || state.Audit[310].ID != "new" || string(state.Settings["new"]) != `"kept"` {
		t.Fatal("non-atomic append")
	}
	if e = s.UpdateOperational(func(st *State) error { st.Audit = nil; return nil }); e != nil {
		t.Fatal(e)
	}
	if len(s.Read().Audit) != 311 {
		t.Fatal("clearing new buffer deleted history")
	}
}
func TestMySQLOperationalConcurrentUpdatesAcrossHosts(t *testing.T) {
	dsn := testsupport.MySQLDSN(t)
	a, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if e = MigrateMySQLAudit(context.Background(), dsn); e != nil {
		t.Fatal(e)
	}
	b, e := OpenMySQL(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for host, s := range []*Store{a, b} {
		wg.Add(1)
		go func(host int, s *Store) {
			defer wg.Done()
			for i := 0; i < 12; i++ {
				e := s.UpdateOperational(func(st *State) error {
					n := 0
					if raw := st.Settings["counter"]; len(raw) > 0 {
						n, _ = strconv.Atoi(string(raw))
					}
					st.Settings["counter"] = []byte(strconv.Itoa(n + 1))
					st.Audit = append(st.Audit, Event{ID: fmt.Sprintf("host-%d-%d", host, i), TenantID: "42", OccurredAt: Now()})
					return nil
				})
				if e != nil {
					results <- e
					return
				}
			}
			results <- nil
		}(host, s)
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	state := a.Read()
	if string(state.Settings["counter"]) != "24" || len(state.Audit) != 24 {
		t.Fatal("concurrent data loss", len(state.Audit))
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if e = a.UpdateOperationalContext(canceled, func(st *State) error {
		st.Audit = append(st.Audit, Event{ID: "canceled", OccurredAt: Now()})
		return nil
	}); e == nil {
		t.Fatal("canceled update committed")
	}
	if len(b.Read().Audit) != 24 {
		t.Fatal("canceled append persisted")
	}
}
