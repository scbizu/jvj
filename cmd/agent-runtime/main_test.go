package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCmdConfigPathRequired(t *testing.T) {
	cmd := newRunCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when config path is missing")
	}
}

func TestRunCmdConfigFlagWorks(t *testing.T) {
	cmd := newRunCmd()
	cmd.SetArgs([]string{"--config", "config.toml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCmdPrintsPlaceholder(t *testing.T) {
	cmd := newVersionCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "dev") {
		t.Fatalf("expected placeholder version, got: %q", buf.String())
	}
}
