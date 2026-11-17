package applications

import (
	"context"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"testing"
)

func TestMySQLGrantSchemaCreateAndValidate(t *testing.T) {
	ctx := context.Background()
	db := preflightDB(t, testsupport.MySQLDSN(t))
	if err := createMySQLGrantSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := validateMySQLGrantSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
}