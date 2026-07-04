package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunNoArgsShowsUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run(nil, &out, &errOut)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "usage: x127") {
		t.Fatalf("stderr = %q, want usage text", errOut.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "x127 v0.1.0-dev") {
		t.Fatalf("stdout = %q, want version", out.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"bogus"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command message", errOut.String())
	}
}
