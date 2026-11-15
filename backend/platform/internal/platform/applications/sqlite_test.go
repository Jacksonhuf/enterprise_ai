package applications

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestSQLiteRestartRollbackAndIndependentReaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	a, e := OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	b, e := OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	id := "channel-replenishment"
	if e = a.Update(func(st *State) error {
		x := st.Apps[id]
		x.Draft.Basic.Name = "跨连接可见"
		st.Apps[id] = x
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if b.Read().Apps[id].Draft.Basic.Name != "跨连接可见" {
		t.Fatal("stale cached grant/config")
	}
	if e = a.Update(func(st *State) error { st.Apps = map[string]App{}; return errors.New("rollback") }); e == nil {
		t.Fatal("expected reject")
	}
	if len(b.Read().Apps) == 0 {
		t.Fatal("rejected update was committed")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store := a
			if i%2 == 1 {
				store = b
			}
			if e := store.Update(func(st *State) error { st.Settings[fmt.Sprint(i)] = json.RawMessage(`"written"`); return nil }); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	if len(a.Read().Settings) != 20 {
		t.Fatal("concurrent updates lost")
	}
	a.Close()
	reopened, e := OpenSQLite(path)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if len(reopened.Read().Settings) != 20 {
		t.Fatal("restart lost data")
	}
	if e = a.Check(); e == nil {
		t.Fatal("closed DB healthy")
	}
	if len(a.Read().Apps) != 0 {
		t.Fatal("closed DB must not expose cached authorization")
	}
}
