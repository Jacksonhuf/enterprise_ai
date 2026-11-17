package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGrantPlannerCLIRequiresCheck(t *testing.T) {
	for _, args := range [][]string{nil, {"--apply"}, {"--check", "extra"}} {
		var stdout, stderr bytes.Buffer
		status := run(args, &stdout, &stderr, func(string) string { t.Fatal("unexpected database access"); return "" })
		if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatal("unsafe CLI accepted", args, status)
		}
	}
	var stdout, stderr bytes.Buffer
	status := run([]string{"--check"}, &stdout, &stderr, func(string) string { return "secret@tcp(localhost:3306)/db?multiStatements=true" })
	if status != 1 || stdout.Len() != 0 || strings.Contains(stderr.String(), "secret") {
		t.Fatal("DSN leaked or failure misreported", status)
	}
}

func TestGrantMigrationApplyRequiresExpectSHA(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"--apply"}, &stdout, &stderr, func(string) string { t.Fatal("unexpected database access"); return "" })
	if status != 2 || stdout.Len() != 0 {
		t.Fatal(status)
	}
	stdout.Reset()
	stderr.Reset()
	status = run([]string{"--apply", "--expect-sha256", strings.Repeat("a", 64)}, &stdout, &stderr, func(string) string { return "invalid@tcp(localhost:3306)/db" })
	if status != 1 || stdout.Len() != 0 {
		t.Fatal("apply without DSN should fail", status, stderr.String())
	}
}