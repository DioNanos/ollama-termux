package launch

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCodexArgs(t *testing.T) {
	c := &Codex{}

	tests := []struct {
		name  string
		model string
		args  []string
		want  []string
	}{
		{"with model", "llama3.2", nil, []string{"--profile", "ollama-launch", "--dangerously-bypass-approvals-and-sandbox", "-m", "llama3.2"}},
		{"empty model", "", nil, []string{"--profile", "ollama-launch", "--dangerously-bypass-approvals-and-sandbox"}},
		{"with model and extra args", "qwen3.5:4b", []string{"-p", "myprofile"}, []string{"--profile", "ollama-launch", "--dangerously-bypass-approvals-and-sandbox", "-m", "qwen3.5:4b", "-p", "myprofile"}},
		{"with sandbox flag", "llama3.2", []string{"--sandbox", "workspace-write"}, []string{"--profile", "ollama-launch", "--dangerously-bypass-approvals-and-sandbox", "-m", "llama3.2", "--sandbox", "workspace-write"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.args(tt.model, tt.args)
			if !slices.Equal(got, tt.want) {
				t.Errorf("args(%q, %v) = %v, want %v", tt.model, tt.args, got, tt.want)
			}
		})
	}
}

func TestCheckCodexVersionSupportsTermuxNodeShim(t *testing.T) {
	tmpDir := t.TempDir()

	nodePath := filepath.Join(tmpDir, "node")
	nodeScript := "#!/bin/sh\nif [ \"$2\" = \"--version\" ]; then\n  echo 'codex-cli 0.120.0-termux'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(nodePath, []byte(nodeScript), 0o755); err != nil {
		t.Fatal(err)
	}

	codexPath := filepath.Join(tmpDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", tmpDir)

	spec, err := (&Codex{}).findCommand()
	if err != nil {
		t.Fatalf("findCommand() error = %v", err)
	}

	if err := checkCodexVersion(spec); err != nil {
		t.Fatalf("checkCodexVersion() error = %v", err)
	}
}

func TestCodexFindCommandFallsBackToTermuxPackageEntrypoint(t *testing.T) {
	tmpDir := t.TempDir()
	packageCLI := filepath.Join(tmpDir, "lib", "node_modules", "@mmmbuto", "codex-cli-termux", "bin", "codex.js")
	if err := os.MkdirAll(filepath.Dir(packageCLI), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packageCLI, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	nodePath := filepath.Join(tmpDir, "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", tmpDir)
	t.Setenv("PREFIX", tmpDir)

	spec, err := (&Codex{}).findCommand()
	if err != nil {
		t.Fatalf("findCommand() error = %v", err)
	}
	if spec.Path != packageCLI {
		t.Fatalf("Path = %q, want %q", spec.Path, packageCLI)
	}
}

func TestWriteCodexProfile(t *testing.T) {
	t.Run("creates new file when none exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		if err := writeCodexProfile(configPath); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		content := string(data)
		if !strings.Contains(content, "[profiles.ollama-launch]") {
			t.Error("missing [profiles.ollama-launch] header")
		}
		if !strings.Contains(content, "openai_base_url") {
			t.Error("missing openai_base_url key")
		}
		if !strings.Contains(content, "/v1/") {
			t.Error("missing /v1/ suffix in base URL")
		}
		if !strings.Contains(content, `forced_login_method = "api"`) {
			t.Error("missing forced_login_method key")
		}
		if !strings.Contains(content, `model_provider = "ollama-launch"`) {
			t.Error("missing model_provider key")
		}
		if !strings.Contains(content, "[model_providers.ollama-launch]") {
			t.Error("missing [model_providers.ollama-launch] section")
		}
		if !strings.Contains(content, `name = "Ollama"`) {
			t.Error("missing model provider name")
		}
	})

	t.Run("appends profile to existing file without profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")
		existing := "[some_other_section]\nkey = \"value\"\n"
		os.WriteFile(configPath, []byte(existing), 0o644)

		if err := writeCodexProfile(configPath); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(configPath)
		content := string(data)

		if !strings.Contains(content, "[some_other_section]") {
			t.Error("existing section was removed")
		}
		if !strings.Contains(content, "[profiles.ollama-launch]") {
			t.Error("missing [profiles.ollama-launch] header")
		}
	})

	t.Run("replaces existing profile section", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")
		existing := "[profiles.ollama-launch]\nopenai_base_url = \"http://old:1234/v1/\"\n\n[model_providers.ollama-launch]\nname = \"Ollama\"\nbase_url = \"http://old:1234/v1/\"\n"
		os.WriteFile(configPath, []byte(existing), 0o644)

		if err := writeCodexProfile(configPath); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(configPath)
		content := string(data)

		if strings.Contains(content, "old:1234") {
			t.Error("old URL was not replaced")
		}
		if strings.Count(content, "[profiles.ollama-launch]") != 1 {
			t.Errorf("expected exactly one [profiles.ollama-launch] section, got %d", strings.Count(content, "[profiles.ollama-launch]"))
		}
		if strings.Count(content, "[model_providers.ollama-launch]") != 1 {
			t.Errorf("expected exactly one [model_providers.ollama-launch] section, got %d", strings.Count(content, "[model_providers.ollama-launch]"))
		}
	})

	t.Run("replaces profile while preserving following sections", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")
		existing := "[profiles.ollama-launch]\nopenai_base_url = \"http://old:1234/v1/\"\n[another_section]\nfoo = \"bar\"\n"
		os.WriteFile(configPath, []byte(existing), 0o644)

		if err := writeCodexProfile(configPath); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(configPath)
		content := string(data)

		if strings.Contains(content, "old:1234") {
			t.Error("old URL was not replaced")
		}
		if !strings.Contains(content, "[another_section]") {
			t.Error("following section was removed")
		}
		if !strings.Contains(content, "foo = \"bar\"") {
			t.Error("following section content was removed")
		}
	})

	t.Run("appends newline to file not ending with newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")
		existing := "[other]\nkey = \"val\""
		os.WriteFile(configPath, []byte(existing), 0o644)

		if err := writeCodexProfile(configPath); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(configPath)
		content := string(data)

		if !strings.Contains(content, "[profiles.ollama-launch]") {
			t.Error("missing [profiles.ollama-launch] header")
		}
		// Should not have double blank lines from missing trailing newline
		if strings.Contains(content, "\n\n\n") {
			t.Error("unexpected triple newline in output")
		}
	})

	t.Run("uses custom OLLAMA_HOST", func(t *testing.T) {
		t.Setenv("OLLAMA_HOST", "http://myhost:9999")
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		if err := writeCodexProfile(configPath); err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(configPath)
		content := string(data)

		if !strings.Contains(content, "myhost:9999/v1/") {
			t.Errorf("expected custom host in URL, got:\n%s", content)
		}
	})
}

func TestEnsureCodexConfig(t *testing.T) {
	t.Run("creates .codex dir and config.toml", func(t *testing.T) {
		tmpDir := t.TempDir()
		setTestHome(t, tmpDir)

		if err := ensureCodexConfig(); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(tmpDir, ".codex", "config.toml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("config.toml not created: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "[profiles.ollama-launch]") {
			t.Error("missing [profiles.ollama-launch] header")
		}
		if !strings.Contains(content, "openai_base_url") {
			t.Error("missing openai_base_url key")
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		setTestHome(t, tmpDir)

		if err := ensureCodexConfig(); err != nil {
			t.Fatal(err)
		}
		if err := ensureCodexConfig(); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(tmpDir, ".codex", "config.toml")
		data, _ := os.ReadFile(configPath)
		content := string(data)

		if strings.Count(content, "[profiles.ollama-launch]") != 1 {
			t.Errorf("expected exactly one [profiles.ollama-launch] section after two calls, got %d", strings.Count(content, "[profiles.ollama-launch]"))
		}
		if strings.Count(content, "[model_providers.ollama-launch]") != 1 {
			t.Errorf("expected exactly one [model_providers.ollama-launch] section after two calls, got %d", strings.Count(content, "[model_providers.ollama-launch]"))
		}
	})
}
