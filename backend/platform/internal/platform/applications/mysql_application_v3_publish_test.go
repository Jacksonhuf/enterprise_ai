package applications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"sync"
	"testing"
	"time"
)

func stageMySQLV3Writers(t *testing.T) (*Store, *sql.DB, string) {
	t.Helper()
	ctx := context.Background()
	dsn := testsupport.MySQLDSN(t)
	s, err := OpenMySQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err = MigrateMySQLAudit(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	db := preflightDB(t, dsn)
	if err = createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = stageMySQLApplicationRoot(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE cognara_fg_state SET schema_version=3 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return s, db, dsn
}

func readV3(t *testing.T, s *Store) State {
	t.Helper()
	st, err := s.readSQLContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestMySQLV3CheckWriterBindsTenantRevisionAndContainer(t *testing.T) {
	ctx := context.Background()
	s, db, _ := stageMySQLV3Writers(t)
	a := readV3(t, s).Apps["channel-replenishment"]
	until := time.Now().Add(5 * time.Minute)
	if _, err := recordMySQLV3Check(ctx, db, "foreign", a.ID, "engineer", a.Revision, "validation", until); !errors.Is(err, errRelationalAppUnavailable) {
		t.Fatal("tenant bypass", err)
	}
	if _, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", "stale", "validation", until); !errors.Is(err, errRelationalDraftConflict) {
		t.Fatal("stale revision accepted", err)
	}
	if _, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, "validation", time.Now().Add(-time.Second)); err == nil {
		t.Fatal("expired check accepted")
	}
	before := readV3(t, s)
	if len(before.Checks) != 0 {
		t.Fatal("unexpected seeded checks")
	}
	if _, err := db.Exec(`UPDATE cognara_fg_application_envelope SET checks_kind='null' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	c, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, "validation-1", until)
	if err != nil {
		t.Fatal(err)
	}
	var kind string
	if err = db.QueryRow(`SELECT checks_kind FROM cognara_fg_application_envelope WHERE id=1`).Scan(&kind); err != nil || kind != "object" {
		t.Fatal("null check container not materialized", kind, err)
	}
	after := readV3(t, s)
	got := after.Checks[c.ID]
	if got.AppID != a.ID || got.Revision != a.Revision || got.UserID != "engineer" || got.ValidationID != "validation-1" || !got.ExpiresAt.Equal(c.ExpiresAt) {
		t.Fatal("check not persisted", got)
	}
	if len(after.Audit) != len(before.Audit) {
		t.Fatal("check recording must not audit, matching the legacy path")
	}
	next := a.Draft
	next.Basic.Name = "edited"
	if _, err = saveMySQLV3Draft(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, next); err != nil {
		t.Fatal(err)
	}
	if _, err = recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, "validation-2", until); !errors.Is(err, errRelationalDraftConflict) {
		t.Fatal("check bound to superseded draft", err)
	}
}

func TestMySQLV3PublishConsumesCheckAtomicallyAndKeepsSnapshot(t *testing.T) {
	ctx := context.Background()
	s, db, dsn := stageMySQLV3Writers(t)
	a := readV3(t, s).Apps["channel-replenishment"]
	var draft []byte
	if err := db.QueryRow(`SELECT raw_payload FROM cognara_fg_application_drafts WHERE app_id=?`, a.ID).Scan(&draft); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(draft, &fields)
	fields["future"] = json.RawMessage(`{"n":9007199254740993,"decimal":0.1234567890123456789}`)
	draft, _ = json.Marshal(fields)
	if _, err := db.Exec(`UPDATE cognara_fg_application_drafts SET config=?,raw_payload=? WHERE app_id=?`, string(draft), draft, a.ID); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(5 * time.Minute)
	c, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, "validation", until)
	if err != nil {
		t.Fatal(err)
	}
	other := "other-release"
	for name, attempt := range map[string]func() error{
		"tenant": func() error {
			_, e := publishMySQLV3Release(ctx, db, "foreign", a.ID, "engineer", "", "", c.ID, a.Revision, a.CurrentReleaseID, "")
			return e
		},
		"actor": func() error {
			_, e := publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "intruder", "", "", c.ID, a.Revision, a.CurrentReleaseID, "")
			return e
		},
		"revision": func() error {
			_, e := publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", c.ID, "stale", a.CurrentReleaseID, "")
			return e
		},
		"current": func() error {
			_, e := publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", c.ID, a.Revision, &other, "")
			return e
		},
		"first": func() error {
			_, e := publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", c.ID, a.Revision, nil, "")
			return e
		},
	} {
		want := errRelationalCheckInvalid
		switch name {
		case "tenant":
			want = errRelationalAppUnavailable
		case "current", "first":
			want = errRelationalReleaseConflict
		}
		if e := attempt(); !errors.Is(e, want) {
			t.Fatal(name, "precondition not enforced", e)
		}
	}
	before := readV3(t, s)
	if _, ok := before.Checks[c.ID]; !ok {
		t.Fatal("rejected publish consumed the check")
	}
	second := preflightDB(t, dsn)
	results := make(chan error, 2)
	releases := make(chan Release, 2)
	var wg sync.WaitGroup
	for _, connection := range []*sql.DB{db, second} {
		wg.Add(1)
		go func(conn *sql.DB) {
			defer wg.Done()
			r, e := publishMySQLV3Release(ctx, conn, a.TenantID, a.ID, "engineer", "client-1", "request-1", c.ID, a.Revision, a.CurrentReleaseID, "第二版")
			results <- e
			if e == nil {
				releases <- r
			}
		}(connection)
	}
	wg.Wait()
	close(results)
	close(releases)
	success, consumed := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, errRelationalCheckInvalid) {
			consumed++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || consumed != 1 {
		t.Fatal("one check published more than once", success, consumed)
	}
	r := <-releases
	after := readV3(t, s)
	published := after.Apps[a.ID]
	if published.CurrentReleaseID == nil || *published.CurrentReleaseID != r.ID || len(published.Releases) != len(a.Releases)+1 || published.Revision != a.Revision || published.UpdatedAt != a.UpdatedAt {
		t.Fatal("release pointer or history incorrect")
	}
	previous, _ := json.Marshal(a.Releases)
	kept, _ := json.Marshal(published.Releases[:len(a.Releases)])
	if !equalApplicationJSON(previous, kept) {
		t.Fatal("earlier release rewritten")
	}
	last := published.Releases[len(a.Releases)]
	if last.ID != r.ID || last.Note != "第二版" || last.PublishedBy != "engineer" || last.Config.Basic != a.Draft.Basic {
		t.Fatal("release content differs", last)
	}
	if _, ok := after.Checks[c.ID]; ok {
		t.Fatal("check not consumed")
	}
	if len(after.Audit) != len(before.Audit)+1 {
		t.Fatal("publish audit not committed together")
	}
	event := after.Audit[len(after.Audit)-1]
	if event.Action != "application.publish" || event.ResourceID != a.ID || event.RequestID != "request-1" || event.ClientID != "client-1" || event.TenantID != a.TenantID {
		t.Fatal("publish audit fields", event)
	}
	var stored []byte
	var ordinal int
	if err = db.QueryRow(`SELECT raw_payload,ordinal FROM cognara_fg_application_releases WHERE release_id=?`, r.ID).Scan(&stored, &ordinal); err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]json.RawMessage
	json.Unmarshal(stored, &snapshot)
	var config map[string]json.RawMessage
	json.Unmarshal(snapshot["config_snapshot"], &config)
	if ordinal != len(a.Releases) || !equalApplicationJSON(config["future"], fields["future"]) {
		t.Fatal("snapshot lost extension precision or ordinal", ordinal)
	}

	stale, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, "validation", until)
	if err != nil {
		t.Fatal(err)
	}
	next := a.Draft
	next.Basic.Name = "after check"
	revision, err := saveMySQLV3Draft(ctx, db, a.TenantID, a.ID, "engineer", a.Revision, next)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", stale.ID, revision, &r.ID, ""); !errors.Is(err, errRelationalCheckInvalid) {
		t.Fatal("check for an older draft published the newer draft", err)
	}
	brief, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", revision, "validation", time.Now().Add(1200*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1300 * time.Millisecond)
	if _, err = publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", brief.ID, revision, &r.ID, ""); !errors.Is(err, errRelationalCheckInvalid) {
		t.Fatal("expired check accepted", err)
	}

	valid, err := recordMySQLV3Check(ctx, db, a.TenantID, a.ID, "engineer", revision, "validation", until)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_publish_audit BEFORE INSERT ON cognara_fg_audit FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected publish failure'`); err != nil {
		t.Fatal(err)
	}
	snapshotBefore := readV3(t, s)
	if out, e := publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", valid.ID, revision, &r.ID, ""); e == nil || out.ID != "" {
		t.Fatal("failed publish returned a release")
	}
	x, _ := json.Marshal(snapshotBefore)
	y, _ := json.Marshal(readV3(t, s))
	if !equalApplicationJSON(x, y) {
		t.Fatal("partial publish persisted")
	}
	if _, err = db.Exec(`DROP TRIGGER fail_publish_audit`); err != nil {
		t.Fatal(err)
	}
	if _, err = publishMySQLV3Release(ctx, db, a.TenantID, a.ID, "engineer", "", "", valid.ID, revision, &r.ID, ""); err != nil {
		t.Fatal("check was lost by the rolled-back publish", err)
	}

	blank, err := createMySQLV3Application(ctx, db, a.TenantID, "engineer", Config{Basic: Basic{Name: "未发布"}}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE cognara_fg_applications SET releases_is_null=1 WHERE app_id=?`, blank.ID); err != nil {
		t.Fatal(err)
	}
	if readV3(t, s).Apps[blank.ID].Releases != nil {
		t.Fatal("null releases container not staged")
	}
	first, err := recordMySQLV3Check(ctx, db, a.TenantID, blank.ID, "engineer", blank.Revision, "validation", until)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = publishMySQLV3Release(ctx, db, a.TenantID, blank.ID, "engineer", "", "", first.ID, blank.Revision, &r.ID, ""); !errors.Is(err, errRelationalReleaseConflict) {
		t.Fatal("unpublished app accepted a current release expectation", err)
	}
	initial, err := publishMySQLV3Release(ctx, db, a.TenantID, blank.ID, "engineer", "", "", first.ID, blank.Revision, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := readV3(t, s).Apps[blank.ID]; len(got.Releases) != 1 || *got.CurrentReleaseID != initial.ID {
		t.Fatal("first publication of null container incorrect")
	}

	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := preflightDB(t, dsn)
	tx, err := reopened.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	state, _, err := readMySQLApplicationV3(ctx, tx, true)
	if err != nil || len(state.Apps[a.ID].Releases) != len(a.Releases)+2 || *state.Apps[blank.ID].CurrentReleaseID != initial.ID {
		t.Fatal("reopen lost publications", err)
	}
}
