package applications

import (
	"context"
	"database/sql"
	"time"
)

// Internal compatibility bridge for isolated migration recovery tests. Caller
// supplies a RepeatableRead (or stronger) transaction so rows and audit share a
// snapshot. This does not activate schema3 or change the public export command.
// The v2 importer stores state in native MySQL JSON; its numeric normalization
// still applies at that explicit compatibility boundary.
func exportMySQLApplicationCheckpoint(ctx context.Context, tx *sql.Tx) ([]byte, int, error) {
	state, err := loadMySQLApplicationLayout(ctx, tx, time.Now().UTC())
	if err != nil {
		return nil, 0, err
	}
	return exportMySQLV2(ctx, tx, state)
}
