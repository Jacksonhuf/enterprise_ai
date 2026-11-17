package main

import (
	"context"
	"encoding/json"
	"flag"
	app "github.com/Tencent/WeKnora/internal/platform/applications"
	"io"
	"os"
	"strings"
)

func run(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("cognara-migrate-usage", flag.ContinueOnError)
	flags.SetOutput(stderr)
	env := flags.String("mysql-dsn-env", "COGNARA_MYSQL_DSN", "MySQL DSN environment variable; never pass a DSN on the command line")
	check := flags.Bool("check", false, "read-only usage migration plan")
	apply := flags.Bool("apply", false, "apply schema7→schema8 usage split")
	expectSHA := flags.String("expect-sha256", "", "source body SHA-256 from plan; required with --apply")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || (*check == *apply) {
		io.WriteString(stderr, "choose exactly one of --check or --apply; --apply requires --expect-sha256\n")
		return 2
	}
	if *apply && strings.TrimSpace(*expectSHA) == "" {
		io.WriteString(stderr, "usage migration apply requires --expect-sha256 from a fresh --check\n")
		return 2
	}
	dsn := getenv(*env)
	if *check {
		report, err := app.PlanMySQLUsage(context.Background(), dsn)
		if err != nil {
			io.WriteString(stderr, err.Error()+"\n")
			return 1
		}
		if json.NewEncoder(stdout).Encode(report) != nil {
			io.WriteString(stderr, "usage migration plan output failed\n")
			return 1
		}
		return 0
	}
	report, err := app.ApplyMySQLUsage(context.Background(), dsn, *expectSHA)
	if err != nil {
		io.WriteString(stderr, err.Error()+"\n")
		return 1
	}
	if json.NewEncoder(stdout).Encode(report) != nil {
		io.WriteString(stderr, "usage migration apply output failed\n")
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv)) }