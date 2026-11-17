package applications

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/platform/testsupport"
)

func TestSessionSchemaDDLValidates(t *testing.T) {
	ctx := context.Background()
	db := preflightDB(t, testsupport.MySQLDSN(t))
	if err := createMySQLSessionSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := validateMySQLSessionSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
}