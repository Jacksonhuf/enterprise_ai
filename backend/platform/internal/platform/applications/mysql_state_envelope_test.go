package applications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestMySQLStateEnvelopePreservesOnlyOpaqueTopLevelFields(t *testing.T) {
	source := []byte(`{"settings":{"removed":true},"SETTINGS":{"stale":true},"audit":[{"old":true}],"Audit":[{"stale":true}],"future":{"large":9007199254740993,"nested":[null,{"keep":true}]},"Future":null}`)
	canonical, _ := json.Marshal(State{Settings: map[string]json.RawMessage{"new": json.RawMessage(`true`)}})
	merged, e := preserveMySQLStateEnvelope(source, canonical)
	if e != nil {
		t.Fatal(e)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(merged, &object) != nil {
		t.Fatal("invalid envelope")
	}
	if string(object["future"]) != `{"large":9007199254740993,"nested":[null,{"keep":true}]}` || string(object["Future"]) != "null" {
		t.Fatal("opaque value changed", string(merged))
	}
	if _, ok := object["SETTINGS"]; ok {
		t.Fatal("known alias survived")
	}
	if _, ok := object["Audit"]; ok {
		t.Fatal("audit alias survived")
	}
	var st State
	if json.Unmarshal(merged, &st) != nil || len(st.Audit) != 0 || len(st.Settings) != 1 || string(st.Settings["new"]) != "true" {
		t.Fatal("known fields not replaced")
	}
	for _, bad := range []string{"null", "[]", "bad"} {
		if _, e = preserveMySQLStateEnvelope([]byte(bad), canonical); e == nil {
			t.Fatal("bad envelope accepted")
		}
	}
}
func TestMySQLRestoredOpaqueStateSurvivesBusinessWrites(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			source := testsupport.MySQLDSN(t)
			s, e := OpenMySQL(source)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			if e = s.Update(func(st *State) error {
				st.Settings["remove"] = json.RawMessage(`true`)
				st.Audit = []Event{{TenantID: "42", ID: "first", OccurredAt: Now()}}
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			if version == 2 {
				if e = MigrateMySQLAudit(context.Background(), source); e != nil {
					t.Fatal(e)
				}
			}
			db := preflightDB(t, source)
			if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future_extension',CAST(? AS JSON),'$.Future_Extension',CAST(? AS JSON),'$.SETTINGS',JSON_OBJECT('stale',true))`, `{"big":9007199254740993,"items":[null,{"keep":true}]}`, `{"other":true}`); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "source.json")
			report, e := ExportMySQLSnapshot(context.Background(), source, path)
			if e != nil {
				t.Fatal(e)
			}
			target := testsupport.MySQLDSN(t)
			if _, e = ImportMySQLSnapshot(context.Background(), target, path, report.SHA256, version); e != nil {
				t.Fatal(e)
			}
			restored, e := OpenMySQL(target)
			if e != nil {
				t.Fatal(e)
			}
			defer restored.Close()
			targetDB := preflightDB(t, target)
			readOpaque := func() map[string]string {
				var raw []byte
				if e := targetDB.QueryRow(`SELECT body FROM cognara_fg_state WHERE id=1`).Scan(&raw); e != nil {
					t.Fatal(e)
				}
				var object map[string]json.RawMessage
				json.Unmarshal(raw, &object)
				return map[string]string{"future_extension": string(object["future_extension"]), "Future_Extension": string(object["Future_Extension"])}
			}
			expected := readOpaque()
			if e = restored.UpdateOperational(func(st *State) error {
				delete(st.Settings, "remove")
				st.Settings["operational"] = json.RawMessage(`true`)
				st.Audit = append(st.Audit, Event{TenantID: "42", ID: "second", OccurredAt: Now()})
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			if e = restored.Update(func(st *State) error {
				st.Settings["full"] = json.RawMessage(`true`)
				st.Audit = append(st.Audit, Event{TenantID: "42", ID: "third", OccurredAt: Now()})
				return nil
			}); e != nil {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(expected, readOpaque()) {
				t.Fatal("business writes erased restored extensions")
			}
			if e = restored.UpdateOperational(func(st *State) error { st.Settings["rollback"] = json.RawMessage(`true`); return errors.New("abort") }); e == nil {
				t.Fatal("failure committed")
			}
			if !reflect.DeepEqual(expected, readOpaque()) {
				t.Fatal("rollback erased extensions")
			}
			if version == 2 {
				if e = restored.UpdateOperational(func(st *State) error {
					st.Settings["rollback"] = json.RawMessage(`true`)
					st.Audit = append(st.Audit, Event{TenantID: "42", ID: "first", OccurredAt: Now()})
					return nil
				}); e == nil {
					t.Fatal("duplicate audit committed")
				}
			}
			state := restored.Read()
			if _, ok := state.Settings["remove"]; ok {
				t.Fatal("deleted setting resurrected")
			}
			if _, ok := state.Settings["rollback"]; ok {
				t.Fatal("rollback persisted")
			}
			if len(state.Audit) != 3 {
				t.Fatal("audit lost")
			}
			var aliases int
			if e = targetDB.QueryRow(`SELECT JSON_CONTAINS_PATH(body,'one','$.SETTINGS') FROM cognara_fg_state WHERE id=1`).Scan(&aliases); e != nil || aliases != 0 {
				t.Fatal("stale alias retained", e)
			}
			again := filepath.Join(t.TempDir(), "after.json")
			if _, e = ExportMySQLSnapshot(context.Background(), target, again); e != nil {
				t.Fatal(e)
			}
			raw, e := os.ReadFile(again)
			if e != nil {
				t.Fatal(e)
			}
			var top map[string]json.RawMessage
			json.Unmarshal(raw, &top)
			if version == 2 {
				json.Unmarshal(top["state"], &top)
			}
			for key, want := range expected {
				var a, b any
				json.Unmarshal(top[key], &a)
				json.Unmarshal([]byte(want), &b)
				if !reflect.DeepEqual(a, b) {
					t.Fatal("re-export lost extension", key)
				}
			}
		})
	}
}
func TestMySQLStateEnvelopeUsesLatestLockedRowAcrossStores(t *testing.T) {
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
	if _, e = db.Exec(`UPDATE cognara_fg_state SET body=JSON_SET(body,'$.future','latest')`); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i, store := range []*Store{a, b} {
		wg.Add(1)
		go func(i int, s *Store) {
			defer wg.Done()
			errs <- s.UpdateOperational(func(st *State) error { st.Settings[fmt.Sprint(i)] = json.RawMessage(`true`); return nil })
		}(i, store)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var value string
	if e = db.QueryRow(`SELECT JSON_UNQUOTE(JSON_EXTRACT(body,'$.future')) FROM cognara_fg_state WHERE id=1`).Scan(&value); e != nil || value != "latest" {
		t.Fatal("cached envelope replaced latest row", e)
	}
	state := a.Read()
	for _, key := range []string{"0", "1"} {
		if string(state.Settings[key]) != "true" {
			t.Fatal("concurrent setting lost")
		}
	}
}
