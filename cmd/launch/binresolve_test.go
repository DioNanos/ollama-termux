package launch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveCommandUsesNodeForNodeShebang(t *testing.T) {
	tmpDir := t.TempDir()
	nodePath := filepath.Join(tmpDir, "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	codexPath := filepath.Join(tmpDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/usr/bin/env node\nconsole.log('hi')\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", tmpDir)

	got, err := resolveCommand("codex")
	if err != nil {
		t.Fatalf("resolveCommand() error = %v", err)
	}
	if got.Path != codexPath {
		t.Fatalf("Path = %q, want %q", got.Path, codexPath)
	}
	if !reflect.DeepEqual(got.Prefix, []string{nodePath}) {
		t.Fatalf("Prefix = %v, want [%q]", got.Prefix, nodePath)
	}
}

func TestResolveCommandFallsBackToTermuxPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	termuxBin := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(termuxBin, 0o755); err != nil {
		t.Fatal(err)
	}

	openURLPath := filepath.Join(termuxBin, "termux-open-url")
	if err := os.WriteFile(openURLPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", t.TempDir())
	t.Setenv("PREFIX", tmpDir)

	got, err := resolveCommand("termux-open-url")
	if err != nil {
		t.Fatalf("resolveCommand() error = %v", err)
	}
	if got.Path != openURLPath {
		t.Fatalf("Path = %q, want %q", got.Path, openURLPath)
	}
	if len(got.Prefix) != 0 {
		t.Fatalf("Prefix = %v, want empty", got.Prefix)
	}
}
