package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed migrations/006_exceptions.sql
var mysqlExceptionsDDL string

func createMySQLExceptionSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(mysqlExceptionsDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func exceptionSchemaSpecs() map[string]applicationSchemaSpec {
	cols := map[string]string{
		"tenant_id": "varbinary(64)", "exception_id": "varbinary(512)", "ordinal": "bigint unsigned",
		"status": "varbinary(32)", "request_id": "varbinary(512)", "metadata": "json", "raw_payload": "longblob",
	}
	return map[string]applicationSchemaSpec{
		"cognara_fg_exceptions": {
			columns: cols,
			indexes: schemaIndexes(
				map[string][]string{
					"PRIMARY":                {"tenant_id", "exception_id"},
					"exception_global_id":      {"exception_id"},
				},
				map[string][]string{
					"exception_tenant_status": {"tenant_id", "status", "ordinal"},
					"exception_request":       {"tenant_id", "request_id"},
				},
			),
			checks: map[string]string{
				"exception_metadata_object": "json_type(metadata)='OBJECT'",
			},
		},
	}
}

func validateMySQLExceptionSchema(ctx context.Context, q applicationSchemaQuery) error {
	for table, spec := range exceptionSchemaSpecs() {
		if err := validateApplicationTable(ctx, q, table, spec); err != nil {
			return err
		}
	}
	return nil
}