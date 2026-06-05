package launch

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTermuxUnsupportedIntegration(t *testing.T) {
	t.Run("off termux everything is supported", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "")
		for _, name := range []string{"claude", "hermes", "openclaw", "codex", "qwen", "pi", "codex-vl"} {
			if err := termuxUnsupportedIntegration(name); err != nil {
				t.Errorf("%s: unexpected error off Termux: %v", name, err)
			}
		}
	})

	t.Run("on termux only the verified set is supported", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "0.118.0")
		for _, name := range termuxIntegrationOrder {
			if err := termuxUnsupportedIntegration(name); err != nil {
				t.Errorf("%s: unexpected error on Termux: %v", name, err)
			}
		}
		for _, name := range []string{"claude", "hermes", "hermes-desktop", "openclaw", "opencode", "cline", "copilot", "droid", "kimi", "pool", "vscode"} {
			if err := termuxUnsupportedIntegration(name); err == nil {
				t.Errorf("%s: expected unsupported error on Termux", name)
			}
		}
	})
}

func TestSupportedMethodsGateOnTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")

	gated := []SupportedIntegration{
		&Claude{}, &Cline{}, &Copilot{}, &Droid{}, &Kimi{},
		&OpenCode{}, &Openclaw{}, &Poolside{}, &Hermes{}, &HermesDesktop{}, &VSCode{},
	}
	for _, runner := range gated {
		if err := runner.Supported(); err == nil {
			t.Errorf("%T: expected Supported() error on Termux", runner)
		}
	}
}

func TestListVisibleIntegrationSpecsOnTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")

	var names []string
	for _, spec := range ListVisibleIntegrationSpecs() {
		names = append(names, spec.Name)
	}

	for _, want := range termuxIntegrationOrder {
		if !slices.Contains(names, want) {
			t.Errorf("expected %s in visible integrations on Termux, got %v", want, names)
		}
	}
	for _, banned := range []string{"claude", "hermes", "hermes-desktop", "openclaw", "opencode", "cline", "copilot", "droid", "pool"} {
		if slices.Contains(names, banned) {
			t.Errorf("did not expect %s in visible integrations on Termux, got %v", banned, names)
		}
	}
}

func TestCodexVLRegistered(t *testing.T) {
	spec, err := LookupIntegrationSpec("codex-vl")
	if err != nil {
		t.Fatalf("codex-vl not registered: %v", err)
	}
	if spec.Runner.String() != "Codex VL" {
		t.Errorf("unexpected runner: %s", spec.Runner.String())
	}
	if spec.Install.EnsureInstalled == nil {
		t.Error("codex-vl must be auto-installable")
	}
	if !slices.Contains(launcherIntegrationOrder, "codex-vl") {
		t.Errorf("codex-vl missing from launcher order %v", launcherIntegrationOrder)
	}
}

func TestQwenExecCommandTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")

	script := filepath.Join(t.TempDir(), "qwen")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("injects yolo approval mode", func(t *testing.T) {
		cmd := qwenExecCommand(script, []string{"--auth-type", "openai"})
		if !slices.Contains(cmd.Args, "--approval-mode") || !slices.Contains(cmd.Args, "yolo") {
			t.Errorf("expected --approval-mode yolo in args, got %v", cmd.Args)
		}
	})

	t.Run("respects explicit approval mode", func(t *testing.T) {
		cmd := qwenExecCommand(script, []string{"--approval-mode", "default"})
		count := 0
		for _, arg := range cmd.Args {
			if arg == "--approval-mode" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected exactly one --approval-mode, got %v", cmd.Args)
		}
	})

	t.Run("off termux no injection", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "")
		cmd := qwenExecCommand(script, []string{"--auth-type", "openai"})
		if slices.Contains(cmd.Args, "--approval-mode") {
			t.Errorf("did not expect --approval-mode off Termux, got %v", cmd.Args)
		}
	})
}

func TestCheckTermuxCodexMinVersion(t *testing.T) {
	cases := []struct {
		raw     string
		min     string
		wantErr bool
	}{
		{"0.136.1-termux", "v0.134.0", false},
		{"0.137.0-termux.1", "v0.134.0", false},
		{"0.136.1", "v0.134.0", false},
		{"0.120.0-termux", "v0.134.0", true},
		{"0.133.9", "v0.134.0", true},
	}
	for _, tc := range cases {
		err := checkTermuxCodexMinVersion(tc.raw, tc.min)
		if (err != nil) != tc.wantErr {
			t.Errorf("checkTermuxCodexMinVersion(%q, %q) error = %v, wantErr %v", tc.raw, tc.min, err, tc.wantErr)
		}
	}
}

func TestCodexArgsTermuxAutoApprove(t *testing.T) {
	t.Run("on termux injects bypass flag", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "0.118.0")
		args, err := (&Codex{}).args("qwen3.5", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
			t.Errorf("expected bypass flag on Termux, got %v", args)
		}
	})

	t.Run("off termux no bypass flag", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "")
		args, err := (&Codex{}).args("qwen3.5", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(args, "--dangerously-bypass-approvals-and-sandbox") {
			t.Errorf("did not expect bypass flag off Termux, got %v", args)
		}
	})
}
