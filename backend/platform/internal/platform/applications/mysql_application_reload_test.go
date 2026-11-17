package applications

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func commitApplicationLayout(t *testing.T, db *sql.DB, raw []byte) {
	t.Helper()
	ctx := context.Background()
	if err := createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = copyMySQLApplicationLayout(ctx, tx, raw, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestMySQLApplicationReloadPreservesNumbersExtensionsAndShapes(t *testing.T) {
	variants := map[string][]byte{"full": rawLayoutFixture(t)}
	for _, kind := range []string{"absent", "null", "object"} {
		for _, shape := range []string{"null", "[]"} {
			var root, apps, app map[string]json.RawMessage
			json.Unmarshal(rawLayoutFixture(t), &root)
			json.Unmarshal(root["apps"], &apps)
			json.Unmarshal(apps["channel-replenishment"], &app)
			app["releases"] = json.RawMessage(shape)
			app["current_release_id"] = json.RawMessage("null")
			apps["channel-replenishment"], _ = json.Marshal(app)
			root["apps"], _ = json.Marshal(apps)
			switch kind {
			case "absent":
				delete(root, "checks")
			case "null":
				root["checks"] = json.RawMessage("null")
			default:
				root["checks"] = json.RawMessage("{}")
			}
			variants[kind+"/"+shape], _ = json.Marshal(root)
		}
	}
	variants["empty"] = []byte(`{"apps":{},"checks":{},"future":{"n":9007199254740993}}`)
	for name, raw := range variants {
		t.Run(name, func(t *testing.T) {
			dsn := testsupport.MySQLDSN(t)
			db := preflightDB(t, dsn)
			commitApplicationLayout(t, db, raw)
			// Discard the original connection pool. Reload has no source/layout argument.
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			reopened := preflightDB(t, dsn)
			tx, err := reopened.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			got, err := loadMySQLApplicationLayout(context.Background(), tx, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if !equalApplicationJSON(raw, got) {
				t.Fatal("database reload lost JSON values or container shape")
			}
			if name == "full" {
				for _, literal := range [][]byte{[]byte("9007199254740993"), []byte("0.1234567890123456789"), []byte("1e25")} {
					if !bytes.Contains(got, literal) {
						t.Fatal("number representation lost")
					}
				}
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMySQLApplicationReloadRejectsCorruptRecords(t *testing.T) {
	db := preflightDB(t, testsupport.MySQLDSN(t))
	commitApplicationLayout(t, db, rawLayoutFixture(t))
	cases := map[string]string{
		"missing-envelope":            `DELETE FROM cognara_fg_application_envelope`,
		"invalid-root":                `UPDATE cognara_fg_application_envelope SET root_payload='[]'`,
		"duplicate-root-authority":    `UPDATE cognara_fg_application_envelope SET root_payload='{"apps":{}}'`,
		"invalid-check-shape":         `UPDATE cognara_fg_application_envelope SET checks_kind='invalid'`,
		"shape-with-records":          `UPDATE cognara_fg_application_envelope SET checks_kind='null'`,
		"missing-draft":               `DELETE FROM cognara_fg_application_drafts`,
		"draft-revision":              `UPDATE cognara_fg_application_drafts SET revision='different'`,
		"draft-json-projection":       `UPDATE cognara_fg_application_drafts SET config='{}'`,
		"draft-raw-invalid":           `UPDATE cognara_fg_application_drafts SET raw_payload='[]'`,
		"owner-projection":            `UPDATE cognara_fg_applications SET owner_id='different'`,
		"agent-projection":            `UPDATE cognara_fg_applications SET agent_id='different'`,
		"updated-projection":          `UPDATE cognara_fg_applications SET updated_at='different'`,
		"current-pointer":             `UPDATE cognara_fg_applications SET current_release_id=NULL`,
		"header-json-projection":      `UPDATE cognara_fg_applications SET metadata='{}'`,
		"header-identity":             `UPDATE cognara_fg_applications SET raw_payload='{}'`,
		"release-shape":               `UPDATE cognara_fg_applications SET releases_is_null=1`,
		"ordinal-gap":                 `UPDATE cognara_fg_application_releases SET ordinal=10 WHERE ordinal=1`,
		"release-json-projection":     `UPDATE cognara_fg_application_releases SET snapshot='{}' WHERE ordinal=1`,
		"release-author-projection":   `UPDATE cognara_fg_application_releases SET published_by='different'`,
		"release-time-projection":     `UPDATE cognara_fg_application_releases SET published_at='different'`,
		"release-metadata-projection": `UPDATE cognara_fg_application_releases SET metadata='{}'`,
		"release-raw-invalid":         `UPDATE cognara_fg_application_releases SET raw_payload='{}'`,
		"check-revision":              `UPDATE cognara_fg_application_checks SET draft_revision='different'`,
		"check-user":                  `UPDATE cognara_fg_application_checks SET user_id='different'`,
		"check-time":                  `UPDATE cognara_fg_application_checks SET expires_at='different'`,
		"check-validation":            `UPDATE cognara_fg_application_checks SET validation_id='different'`,
		"check-metadata":              `UPDATE cognara_fg_application_checks SET metadata='{}'`,
		"check-raw-invalid":           `UPDATE cognara_fg_application_checks SET raw_payload='{}'`,
	}
	for name, statement := range cases {
		t.Run(name, func(t *testing.T) {
			tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			result, err := tx.Exec(statement)
			if err != nil {
				t.Fatal("corruption fixture failed", err)
			}
			if n, err := result.RowsAffected(); err != nil || n == 0 {
				t.Fatal("mutation did not affect records", n, err)
			}
			if _, err = loadMySQLApplicationLayout(context.Background(), tx, time.Now()); err == nil {
				t.Fatal("corruption accepted")
			}
		})
	}
}

func TestMySQLApplicationCopyRejectsRoundedEquivalentRawMutation(t *testing.T) {
	db := preflightDB(t, testsupport.MySQLDSN(t))
	if err := createMySQLApplicationSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	// Both decimals can share a MySQL DOUBLE projection; the persisted raw value
	// must still compare exactly to the original source after database reload.
	if _, err := db.Exec(`CREATE TRIGGER corrupt_raw_precision BEFORE INSERT ON cognara_fg_application_drafts FOR EACH ROW SET NEW.raw_payload=REPLACE(CONVERT(NEW.raw_payload USING utf8mb4),'0.1234567890123456789','0.1234567890123456788')`); err != nil {
		t.Fatal(err)
	}
	if err := rehearseMySQLApplicationCopy(context.Background(), db, rawLayoutFixture(t), time.Now()); err == nil || err.Error() != "application database round-trip reconciliation failed" {
		t.Fatal("expected strict raw reconciliation failure after valid SQL projections", err)
	}
	for table := range applicationSchemaSpecs() {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("partial precision copy", table, count, err)
		}
	}
}
