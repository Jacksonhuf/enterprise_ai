package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed migrations/005_settings.sql
var mysqlSettingsDDL string

func createMySQLSettingsSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(mysqlSettingsDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func settingsSchemaSpecs() map[string]applicationSchemaSpec {
	cols := map[string]string{
		"tenant_id": "varbinary(64)", "setting_key": "varbinary(512)", "owner_id": "varbinary(512)",
		"section": "varbinary(128)", "revision": "varbinary(512)", "metadata": "json", "raw_payload": "json",
	}
	return map[string]applicationSchemaSpec{
		"cognara_fg_settings": {
			columns: cols,
			indexes: schemaIndexes(
				map[string][]string{
					"PRIMARY":             {"tenant_id", "setting_key"},
					"setting_key_global":  {"setting_key"},
				},
				map[string][]string{
					"setting_owner": {"tenant_id", "owner_id", "section"},
				},
			),
			checks: map[string]string{
				"setting_metadata_object": "json_type(metadata)='OBJECT'",
				"setting_payload_json":    "json_type(raw_payload)='OBJECT'",
			},
		},
	}
}

func validateMySQLSettingsSchema(ctx context.Context, q applicationSchemaQuery) error {
	for table, spec := range settingsSchemaSpecs() {
		if err := validateApplicationTable(ctx, q, table, spec); err != nil {
			return err
		}
	}
	return nil
}