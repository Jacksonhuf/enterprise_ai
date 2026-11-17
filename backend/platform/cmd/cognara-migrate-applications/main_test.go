package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPlannerCLIRequiresExplicitReadOnlyMode(t *testing.T) {
	for _, args := range [][]string{nil, {"--apply"}, {"--check", "unexpected"}} {
		var stdout, stderr bytes.Buffer
		status := run(args, &stdout, &stderr, func(string) string { t.Fatal("invalid command attempted database access"); return "" })
		if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("unsafe CLI accepted", args, status)
		}
	}
}
func TestPlannerCLIRedactsInvalidDSN(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"--check"}, &stdout, &stderr, func(string) string { return "secret-password@tcp(localhost:3306)/db?multiStatements=true" })
	if status != 1 || stdout.Len() != 0 || strings.Contains(stderr.String(), "secret-password") || !strings.Contains(stderr.String(), "invalid MySQL") {
		t.Fatal("DSN leaked or failure misreported", status)
	}
}
