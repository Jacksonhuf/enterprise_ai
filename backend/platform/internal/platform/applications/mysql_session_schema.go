package applications

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
)

//go:embed migrations/004_sessions.sql
var mysqlSessionDDL string

func createMySQLSessionSchema(ctx context.Context, db *sql.DB) error {
	for _, statement := range strings.Split(mysqlSessionDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func sessionSchemaSpecs() map[string]applicationSchemaSpec {
	sessionCols := map[string]string{
		"tenant_id": "varbinary(64)", "session_id": "varbinary(512)", "app_id": "varbinary(512)",
		"owner_id": "varbinary(512)", "title_customized": "tinyint(1)", "metadata": "json", "raw_payload": "longblob",
	}
	messageCols := map[string]string{
		"tenant_id": "varbinary(64)", "session_id": "varbinary(512)", "message_id": "varbinary(512)",
		"ordinal": "int unsigned", "metadata": "json", "raw_payload": "longblob",
	}
	return map[string]applicationSchemaSpec{
		"cognara_fg_sessions": {
			columns: sessionCols,
			indexes: schemaIndexes(
				map[string][]string{
					"PRIMARY":           {"tenant_id", "session_id"},
					"session_global_id": {"session_id"},
				},
				map[string][]string{
					"session_owner": {"tenant_id", "owner_id", "app_id"},
					"session_app":   {"tenant_id", "app_id"},
				},
			),
			checks: map[string]string{"session_metadata_object": "json_type(metadata)='OBJECT'"},
		},
		"cognara_fg_session_messages": {
			columns: messageCols,
			indexes: schemaIndexes(
				map[string][]string{
					"PRIMARY":               {"tenant_id", "session_id", "message_id"},
					"session_message_order": {"tenant_id", "session_id", "ordinal"},
				},
				map[string][]string{
					"session_message_global": {"message_id"},
				},
			),
			checks: map[string]string{"session_message_metadata_object": "json_type(metadata)='OBJECT'"},
		},
	}
}

func validateMySQLSessionSchema(ctx context.Context, q applicationSchemaQuery) error {
	for table, spec := range sessionSchemaSpecs() {
		if err := validateApplicationTable(ctx, q, table, spec); err != nil {
			return err
		}
	}
	return nil
}