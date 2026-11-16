package applications

import (
	"context"
	"encoding/json"
)

// Retain the existing exception cursor error identity for callers.
var ErrExceptionPageCursor = ErrGovernancePageCursor

func (s *Store) QueryMySQLExceptions(ctx context.Context, tenant, after string, limit int, match func(map[string]any) bool, idOf func(map[string]any) string) ([]json.RawMessage, bool, bool, error) {
	return s.queryMySQLGovernanceArray(ctx, "exceptions", tenant, after, limit, match, idOf)
}
func (s *Store) QueryMySQLUsage(ctx context.Context, tenant, after string, limit int, match func(map[string]any) bool, idOf func(map[string]any) string) ([]json.RawMessage, bool, bool, error) {
	return s.queryMySQLGovernanceArray(ctx, "usage", tenant, after, limit, match, idOf)
}
