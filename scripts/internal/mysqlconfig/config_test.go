package mysqlconfig

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestBindReadsMYSQLDSNAndLegacyEnvironment(t *testing.T) {
	t.Setenv("MYSQL_DSN", "ops:secret@tcp(db.example.com:3307)/miniblog?parseTime=true")
	t.Setenv("MYSQL_HOST", "legacy-host")
	t.Setenv("MYSQL_PORT", "3308")
	t.Setenv("MYSQL_USERNAME", "legacy-user")
	t.Setenv("MYSQL_PASSWORD", "legacy-password")
	t.Setenv("MYSQL_DATABASE", "legacy-database")

	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	config := Bind(flags)
	if err := flags.Parse([]string{"-host", "flag-host", "-database", "flag-database"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	opts := config.DBOptions(1)
	if opts.DSN != "ops:secret@tcp(db.example.com:3307)/miniblog?parseTime=true" {
		t.Fatalf("DSN = %q", opts.DSN)
	}
	if opts.Host != "flag-host" {
		t.Fatalf("Host = %q, want flag-host", opts.Host)
	}
	if opts.Port != "3308" {
		t.Fatalf("Port = %q, want 3308", opts.Port)
	}
	if opts.Username != "legacy-user" {
		t.Fatalf("Username = %q, want legacy-user", opts.Username)
	}
	if opts.Password != "legacy-password" {
		t.Fatalf("Password = %q, want legacy-password", opts.Password)
	}
	if opts.Database != "flag-database" {
		t.Fatalf("Database = %q, want flag-database", opts.Database)
	}
	if opts.LogLevel != 1 {
		t.Fatalf("LogLevel = %d, want 1", opts.LogLevel)
	}
}

func TestBindUsesLegacyDefaultsWithoutEnvironment(t *testing.T) {
	for _, key := range []string{
		"MYSQL_DSN",
		"MYSQL_HOST",
		"MYSQL_PORT",
		"MYSQL_USERNAME",
		"MYSQL_PASSWORD",
		"MYSQL_DATABASE",
	} {
		t.Setenv(key, "")
	}

	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	opts := Bind(flags).DBOptions(1)

	if opts.DSN != "" {
		t.Fatalf("DSN = %q, want empty", opts.DSN)
	}
	if opts.Host != "localhost" || opts.Port != "3306" {
		t.Fatalf("address = %s:%s, want localhost:3306", opts.Host, opts.Port)
	}
	if opts.Username != "miniblog" || opts.Password != "miniblog123" || opts.Database != "miniblog" {
		t.Fatalf("legacy defaults changed: user=%q password=%q database=%q", opts.Username, opts.Password, opts.Database)
	}
}

func TestHelpAndParseFailureNeverPrintEnvironmentDefaults(t *testing.T) {
	t.Setenv("MYSQL_DSN", "do-not-print-dsn")
	t.Setenv("MYSQL_PASSWORD", "do-not-print-password")
	t.Setenv("MYSQL_USERNAME", "do-not-print-account")
	for _, args := range [][]string{{"-h"}, {"-unknown-option"}} {
		var out bytes.Buffer
		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		flags.SetOutput(&out)
		Bind(flags)
		_ = flags.Parse(args)
		if strings.Contains(out.String(), "do-not-print") {
			t.Fatal("credential environment was printed in usage")
		}
	}
}

func TestEnvironmentIsReadAfterParsingAndEmptyExplicitPasswordWins(t *testing.T) {
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("MYSQL_PASSWORD", "initial")
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	config := Bind(flags)
	t.Setenv("MYSQL_PASSWORD", "late")
	if err := flags.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if config.DBOptions(1).Password != "late" {
		t.Fatal("environment was captured too early")
	}
	if err := flags.Parse([]string{"-db-password", ""}); err != nil {
		t.Fatal(err)
	}
	if config.DBOptions(1).Password != "" {
		t.Fatal("explicit empty credential did not override environment")
	}
}
