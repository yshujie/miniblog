package main

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestAuditCLIPrivateEnvironmentNeverPrinted(t *testing.T) {
	if mode := os.Getenv("MINIBLOG_AUDIT_CLI_TEST_MODE"); mode != "" {
		flag.CommandLine = flag.NewFlagSet("audit-content", flag.ExitOnError)
		switch mode {
		case "help":
			os.Args = []string{"audit-content", "-h"}
		case "invalid":
			os.Args = []string{"audit-content", "-unknown-option"}
		case "connection":
			os.Args = []string{"audit-content"}
		default:
			os.Exit(3)
		}
		os.Exit(run())
	}
	for _, mode := range []string{"help", "invalid", "connection"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestAuditCLIPrivateEnvironmentNeverPrinted$")
			cmd.Env = append(os.Environ(), "MINIBLOG_AUDIT_CLI_TEST_MODE="+mode, "MYSQL_PASSWORD=PRIVATE_PASSWORD_SENTINEL", "MYSQL_DSN=PRIVATE_DSN_SENTINEL", "MINIBLOG_NOTION_TOKEN=PRIVATE_TOKEN_SENTINEL", "MINIBLOG_NOTION_BOOTSTRAP_TOKEN=PRIVATE_BOOTSTRAP_SENTINEL")
			output, err := cmd.CombinedOutput()
			if mode == "help" && err != nil {
				t.Fatalf("help failed: %s", output)
			}
			if mode != "help" && err == nil {
				t.Fatalf("invalid operation succeeded: %s", output)
			}
			for _, sentinel := range []string{"PRIVATE_PASSWORD_SENTINEL", "PRIVATE_DSN_SENTINEL", "PRIVATE_TOKEN_SENTINEL", "PRIVATE_BOOTSTRAP_SENTINEL"} {
				if strings.Contains(string(output), sentinel) {
					t.Fatalf("private environment disclosed in %s", mode)
				}
			}
		})
	}
}
