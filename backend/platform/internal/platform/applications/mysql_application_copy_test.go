package applications

import (
	"context"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
	"time"
)

func TestMySQLApplicationCopyRehearsalAlwaysRollsBack(t *testing.T) {
	db := preflightDB(t, testsupport.MySQLDSN(t))
	ctx := context.Background()
	if err := createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	raw := rawLayoutFixture(t)
	empty := func() {
		t.Helper()
		for table := range applicationSchemaSpecs() {
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
				t.Fatal(table, count, err)
			}
		}
	}
	for i := 0; i < 2; i++ {
		if err := rehearseMySQLApplicationCopy(ctx, db, raw, time.Now()); err != nil {
			t.Fatal(err)
		}
		empty()
	}
	// Fail only after parent, draft and releases have been inserted.
	if _, err := db.Exec(`CREATE TRIGGER reject_copy_check BEFORE INSERT ON cognara_fg_application_checks FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected copy failure'`); err != nil {
		t.Fatal(err)
	}
	if err := rehearseMySQLApplicationCopy(ctx, db, raw, time.Now()); err == nil {
		t.Fatal("injected failure accepted")
	}
	empty()
	if _, err := db.Exec(`DROP TRIGGER reject_copy_check`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER corrupt_copy_draft BEFORE INSERT ON cognara_fg_application_drafts FOR EACH ROW SET NEW.revision='corrupted'`); err != nil {
		t.Fatal(err)
	}
	if err := rehearseMySQLApplicationCopy(ctx, db, raw, time.Now()); err == nil {
		t.Fatal("corrupted projection accepted")
	}
	empty()
	if _, err := db.Exec(`DROP TRIGGER corrupt_copy_draft`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cognara_fg_applications(tenant_id,app_id,agent_id,owner_id,current_release_id,updated_at,metadata,releases_is_null,raw_payload) VALUES('42','existing','agent','owner',NULL,'now','{}',0,'{}')`); err != nil {
		t.Fatal(err)
	}
	if err := rehearseMySQLApplicationCopy(ctx, db, raw, time.Now()); err == nil {
		t.Fatal("nonempty target accepted")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cognara_fg_applications WHERE app_id='existing'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("existing target changed", count, err)
	}
}
