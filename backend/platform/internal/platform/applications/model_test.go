package applications

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPersistenceIsolationAndRollback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	id := "channel-replenishment"
	snapshot := s.Read()
	a := snapshot.Apps[id]
	a.Draft.Basic.Name = "leaked"
	snapshot.Apps[id] = a
	if s.Read().Apps[id].Draft.Basic.Name == "leaked" {
		t.Fatal("read exposed state")
	}
	err := s.Update(func(st *State) error {
		a := st.Apps[id]
		a.Draft.Basic.Name = "rollback"
		st.Apps[id] = a
		return errors.New("reject")
	})
	if err == nil || s.Read().Apps[id].Draft.Basic.Name == "rollback" {
		t.Fatal("failed transaction committed")
	}
	if e = s.Update(func(st *State) error { a := st.Apps[id]; a.Draft.Basic.Name = "persisted"; st.Apps[id] = a; return nil }); e != nil {
		t.Fatal(e)
	}
	reopened, e := Open(path)
	if e != nil || reopened.Read().Apps[id].Draft.Basic.Name != "persisted" {
		t.Fatal("restart lost state", e)
	}
}
