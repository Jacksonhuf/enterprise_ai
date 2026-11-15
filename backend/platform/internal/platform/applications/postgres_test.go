package applications

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPostgresConcurrentUpdatesRollbackAndReopen(t *testing.T) {
	dsn := testsupport.PostgresDSN(t)
	a, e := OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	id := "channel-replenishment"
	if e = a.Update(func(st *State) error {
		app := st.Apps[id]
		app.Draft.Basic.Name = "跨连接可见"
		st.Apps[id] = app
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if b.Read().Apps[id].Draft.Basic.Name != "跨连接可见" {
		t.Fatal("Postgres returned cached state")
	}
	if e = a.Update(func(st *State) error { st.Apps = map[string]App{}; return errors.New("reject") }); e == nil || len(b.Read().Apps) != 1 {
		t.Fatal("rollback failed")
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store := a
			if i%2 == 1 {
				store = b
			}
			if e := store.Update(func(st *State) error {
				var n int
				if raw := st.Settings["counter"]; len(raw) > 0 {
					if e := json.Unmarshal(raw, &n); e != nil {
						return e
					}
				}
				st.Settings["counter"], _ = json.Marshal(n + 1)
				return nil
			}); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	var count int
	json.Unmarshal(b.Read().Settings["counter"], &count)
	if count != 32 {
		t.Fatalf("lost updates: %d", count)
	}
	a.Close()
	reopened, e := OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	json.Unmarshal(reopened.Read().Settings["counter"], &count)
	if count != 32 || reopened.Read().Apps[id].Draft.Basic.Name != "跨连接可见" {
		t.Fatal("reopen reseeded existing state")
	}
}
func TestPostgresLockCancellationReleasesTransaction(t *testing.T) {
	dsn := testsupport.PostgresDSN(t)
	a, e := OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	lock, e := a.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Rollback()
	if _, e = lock.Exec(`SELECT id FROM cognara_fg_state WHERE id=1 FOR UPDATE`); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	called := false
	e = b.UpdateContext(ctx, func(st *State) error { called = true; return nil })
	if e == nil || called {
		t.Fatal("blocked canceled transaction executed mutation")
	}
	if !errors.Is(e, context.DeadlineExceeded) && !strings.Contains(e.Error(), "context deadline exceeded") {
		t.Fatalf("unexpected cancellation error: %v", e)
	}
	lock.Rollback()
	if e = b.Update(func(st *State) error { st.Settings["after-cancel"] = json.RawMessage(`true`); return nil }); e != nil {
		t.Fatal("canceled transaction retained row lock", e)
	}
}
func TestPostgresBootstrapConcurrentAndVersionGuard(t *testing.T) {
	dsn := testsupport.PostgresDSN(t)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, e := OpenPostgres(dsn)
			if e != nil {
				t.Error(e)
				return
			}
			defer s.Close()
			if len(s.Read().Apps) != 1 {
				t.Error("bootstrap did not preserve a single initial aggregate")
			}
		}()
	}
	wg.Wait()
	s, e := OpenPostgres(dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.db.Exec(`UPDATE cognara_fg_state SET schema_version=99 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	if e = s.Check(); e == nil {
		t.Fatal("unsupported schema reported healthy")
	}
	if len(s.Read().Apps) != 0 {
		t.Fatal("unsupported schema exposed cached data")
	}
	if other, e := OpenPostgres(dsn); e == nil {
		other.Close()
		t.Fatal("unsupported schema accepted")
	}
	var version int
	s.db.QueryRow(`SELECT schema_version FROM cognara_fg_state WHERE id=1`).Scan(&version)
	if version != 99 {
		t.Fatal("bootstrap overwrote unknown schema")
	}
}
func TestPostgresConfigurationErrorsRedactCredentials(t *testing.T) {
	for _, dsn := range []string{"", "postgres://secret-user:secret-key%xx@invalid/db"} {
		if s, e := OpenPostgres(dsn); e == nil {
			s.Close()
			t.Fatal("invalid DSN accepted")
		} else if strings.Contains(e.Error(), "secret-key") || strings.Contains(e.Error(), "secret-user") {
			t.Fatal("DSN leaked")
		}
	}
}
