package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed migrations/007_usage.sql
var mysqlUsageDDL string

func createMySQLUsageSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(mysqlUsageDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func usageSchemaSpecs() map[string]applicationSchemaSpec {
	cols := map[string]string{
		"tenant_id": "varbinary(64)", "event_id": "varbinary(512)", "ordinal": "bigint unsigned",
		"unit": "varbinary(32)", "capability": "varbinary(512)", "occurred_at": "varbinary(64)",
		"metadata": "json", "raw_payload": "longblob",
	}
	return map[string]applicationSchemaSpec{
		"cognara_fg_usage": {
			columns: cols,
			indexes: schemaIndexes(
				map[string][]string{
					"PRIMARY":          {"tenant_id", "event_id"},
					"usage_global_id": {"event_id"},
				},
				map[string][]string{
					"usage_tenant_unit":       {"tenant_id", "unit", "ordinal"},
					"usage_tenant_capability": {"tenant_id", "capability", "ordinal"},
				},
			),
			checks: map[string]string{
				"usage_metadata_object": "json_type(metadata)='OBJECT'",
			},
		},
	}
}

func validateMySQLUsageSchema(ctx context.Context, q applicationSchemaQuery) error {
	for table, spec := range usageSchemaSpecs() {
		if err := validateApplicationTable(ctx, q, table, spec); err != nil {
			return err
		}
	}
	return nil
}