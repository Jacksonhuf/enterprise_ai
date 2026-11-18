package api_test

import "testing"

// TestR20Schema8RunbookDefaultSmoke is the canonical post-apply smoke gate referenced
// by mysql-schema8-production-apply-runbook.md (write-then-read bundle on dual hosts).
func TestR20Schema8RunbookDefaultSmoke(t *testing.T) {
	TestR20Schema8MixedSmokeWriteThenReadOrchestration(t)
}