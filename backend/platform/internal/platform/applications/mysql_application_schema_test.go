package applications

import (
	"context"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"strings"
	"testing"
)

func TestApplicationCheckNormalizationPreservesLiteralSemantics(t *testing.T) {
	if normalizeApplicationCheck("(json_type(`config`) = "+`_utf8mb4\'OBJECT\'`+")") != normalizeApplicationCheck("json_type(config)='OBJECT'") {
		t.Fatal("captured MySQL escaped literal rejected")
	}
	if normalizeApplicationCheck("json_type(config)="+`_utf8mb4\'OBJECT\'`+" OR 1") == normalizeApplicationCheck("json_type(config)='OBJECT'") {
		t.Fatal("weakened captured expression accepted")
	}
	if normalizeApplicationCheck("(JSON_TYPE(`config`) = _utf8mb4'OBJECT')") != normalizeApplicationCheck("json_type(config)='OBJECT'") {
		t.Fatal("MySQL formatting rejected")
	}
	for _, clause := range []string{"json_type(config)='OB JECT'", "json_type(config)='OBJ(ECT)'", "json_type(config)='OB_utf8mb4JECT'", "json_type(config)='object'", "json_type(config)='OBJECT' OR 1"} {
		if normalizeApplicationCheck(clause) == normalizeApplicationCheck("json_type(config)='OBJECT'") {
			t.Fatal("changed constraint accepted", clause)
		}
	}
}
func TestMySQLApplicationSchemaAndTenantForeignKeys(t *testing.T) {
	db := preflightDB(t, testsupport.MySQLDSN(t))
	ctx := context.Background()
	if err := createMySQLApplicationSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := validateMySQLApplicationSchema(ctx, db); err != nil {
		rows, e := db.Query(`SELECT CONSTRAINT_NAME,CHECK_CLAUSE FROM information_schema.CHECK_CONSTRAINTS WHERE CONSTRAINT_SCHEMA=DATABASE()`)
		if e == nil {
			for rows.Next() {
				var name, clause string
				if e = rows.Scan(&name, &clause); e != nil {
					break
				}
				t.Logf("DDL CHECK %s: %s [normalized=%s]", name, clause, normalizeApplicationCheck(clause))
			}
			rows.Close()
		}
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateMySQLApplicationSchema(ctx, tx); err != nil {
		tx.Rollback()
		t.Fatal("validator must support same-transaction queries", err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	for _, row := range [][2]string{{"42", "App"}, {"43", "app"}} {
		if _, err = db.Exec(`INSERT INTO cognara_fg_applications(tenant_id,app_id,agent_id,owner_id,current_release_id,updated_at,metadata,releases_is_null) VALUES(?,?, 'agent','owner',NULL,'2026-11-17T00:00:00.123456789Z','{}',0)`, row[0], row[1]); err != nil {
			t.Fatal("binary identifiers or parent insert", err)
		}
	}
	if _, err = db.Exec(`INSERT INTO cognara_fg_application_drafts VALUES('43','App','revision','{}')`); err == nil {
		t.Fatal("cross-tenant draft inserted")
	}
	if _, err = db.Exec(`INSERT INTO cognara_fg_application_drafts VALUES('42','App','revision','[]')`); err == nil {
		t.Fatal("non-object config inserted")
	}
	if _, err = db.Exec(`INSERT INTO cognara_fg_application_releases VALUES('42','App','release',1,'{}','2026-11-17T00:00:00Z','owner','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE cognara_fg_applications SET current_release_id='release' WHERE app_id='App'`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE cognara_fg_applications SET current_release_id='release' WHERE app_id='app'`); err == nil {
		t.Fatal("cross-tenant release pointer accepted")
	}
	if _, err = db.Exec(`DELETE FROM cognara_fg_application_releases WHERE release_id='release'`); err == nil {
		t.Fatal("current release reference deleted")
	}
	if _, err = db.Exec(`UPDATE cognara_fg_applications SET releases_is_null=2`); err == nil {
		t.Fatal("invalid collection shape accepted")
	}
}
func TestMySQLApplicationSchemaRejectsIncompatibleLayouts(t *testing.T) {
	for name, statement := range map[string]string{
		"missing-table":   "DROP TABLE cognara_fg_application_drafts",
		"extra-column":    "ALTER TABLE cognara_fg_application_drafts ADD extension INT NULL",
		"default-column":  "ALTER TABLE cognara_fg_application_drafts ALTER COLUMN revision SET DEFAULT 'fallback'",
		"cascade-fk":      "ALTER TABLE cognara_fg_application_drafts DROP FOREIGN KEY draft_application_fk; ALTER TABLE cognara_fg_application_drafts ADD CONSTRAINT draft_application_fk FOREIGN KEY(tenant_id,app_id) REFERENCES cognara_fg_applications(tenant_id,app_id) ON UPDATE RESTRICT ON DELETE CASCADE",
		"wrong-column":    "ALTER TABLE cognara_fg_application_drafts MODIFY revision VARBINARY(64) NOT NULL",
		"missing-fk":      "ALTER TABLE cognara_fg_application_drafts DROP FOREIGN KEY draft_application_fk",
		"disabled-check":  "ALTER TABLE cognara_fg_application_drafts ALTER CHECK draft_config_object NOT ENFORCED",
		"weak-check":      "ALTER TABLE cognara_fg_application_drafts DROP CHECK draft_config_object, ADD CONSTRAINT draft_config_object CHECK(JSON_VALID(config))",
		"invisible-index": "ALTER TABLE cognara_fg_applications ALTER INDEX application_owner INVISIBLE",
		"prefix-index":    "ALTER TABLE cognara_fg_applications DROP INDEX application_owner, ADD KEY application_owner(tenant_id,owner_id(20),app_id)",
		"wrong-collation": "ALTER TABLE cognara_fg_application_drafts DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci",
	} {
		t.Run(name, func(t *testing.T) {
			db := preflightDB(t, testsupport.MySQLDSN(t))
			ctx := context.Background()
			if err := createMySQLApplicationSchema(ctx, db); err != nil {
				t.Fatal(err)
			}
			if err := validateMySQLApplicationSchema(ctx, db); err != nil {
				t.Fatal("valid baseline rejected before mutation", err)
			}
			for _, part := range strings.Split(statement, ";") {
				if _, err := db.Exec(part); err != nil {
					t.Fatal("fixture mutation failed", err)
				}
			}
			if err := validateMySQLApplicationSchema(ctx, db); err == nil {
				t.Fatal("incompatible schema accepted")
			}
		})
	}
}
func TestMySQLApplicationDDLDoesNotReplaceExistingTables(t *testing.T) {
	db := preflightDB(t, testsupport.MySQLDSN(t))
	if _, err := db.Exec(`CREATE TABLE cognara_fg_applications(id INT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := createMySQLApplicationSchema(context.Background(), db); err == nil {
		t.Fatal("existing target overwritten")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE()`).Scan(&count); err != nil || count != 1 {
		t.Fatal("DDL continued after collision", count, err)
	}
}
