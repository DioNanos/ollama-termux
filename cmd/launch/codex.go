package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ollama/ollama/envconfig"
	"golang.org/x/mod/semver"
)

// Codex implements Runner for Codex integration
type Codex struct{}

func (c *Codex) String() string { return "Codex" }

const codexProfileName = "ollama-launch"

func (c *Codex) args(model string, extra []string) []string {
	args := []string{"--profile", codexProfileName, "--dangerously-bypass-approvals-and-sandbox"}
	if model != "" {
		args = append(args, "-m", model)
	}
	args = append(args, extra...)
	return args
}

func (c *Codex) Run(model string, args []string) error {
	codex, err := c.findCommand()
	if err != nil {
		return fmt.Errorf("codex is not installed\n\nInstall with:\n  npm install -g @mmmbuto/codex-cli-termux  (recommended on Termux)\n  or\n  npm install -g @openai/codex")
	}

	if err := checkCodexVersion(codex); err != nil {
		return err
	}

	if err := ensureCodexConfig(); err != nil {
		return fmt.Errorf("failed to configure codex: %w", err)
	}

	cmd := codex.Command(c.args(model, args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=ollama",
	)
	return cmd.Run()
}

func (c *Codex) findCommand() (resolvedCommand, error) {
	termuxPackageCLI := termuxPackageEntrypoint("@mmmbuto/codex-cli-termux", "bin/codex.js")
	return resolveCommand("codex", termuxPackageCLI)
}

// ensureCodexConfig writes a [profiles.ollama-launch] section to ~/.codex/config.toml
// with openai_base_url pointing to the local Ollama server.
func ensureCodexConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(codexDir, "config.toml")
	return writeCodexProfile(configPath)
}

// writeCodexProfile ensures ~/.codex/config.toml has the ollama-launch profile
// and model provider sections with the correct base URL.
func writeCodexProfile(configPath string) error {
	baseURL := envconfig.Host().String() + "/v1/"

	sections := []struct {
		header string
		lines  []string
	}{
		{
			header: fmt.Sprintf("[profiles.%s]", codexProfileName),
			lines: []string{
				fmt.Sprintf("openai_base_url = %q", baseURL),
				`forced_login_method = "api"`,
				fmt.Sprintf("model_provider = %q", codexProfileName),
			},
		},
		{
			header: fmt.Sprintf("[model_providers.%s]", codexProfileName),
			lines: []string{
				`name = "Ollama"`,
				fmt.Sprintf("base_url = %q", baseURL),
			},
		},
	}

	content, readErr := os.ReadFile(configPath)
	text := ""
	if readErr == nil {
		text = string(content)
	}

	for _, s := range sections {
		block := strings.Join(append([]string{s.header}, s.lines...), "\n") + "\n"

		if idx := strings.Index(text, s.header); idx >= 0 {
			// Replace the existing section up to the next section header.
			rest := text[idx+len(s.header):]
			if endIdx := strings.Index(rest, "\n["); endIdx >= 0 {
				text = text[:idx] + block + rest[endIdx+1:]
			} else {
				text = text[:idx] + block
			}
		} else {
			// Append the section.
			if text != "" && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			if text != "" {
				text += "\n"
			}
			text += block
		}
	}

	return os.WriteFile(configPath, []byte(text), 0o644)
}

func checkCodexVersion(codex resolvedCommand) error {
	out, err := codex.Command("--version").Output()
	if err != nil {
		return fmt.Errorf("failed to get codex version: %w", err)
	}

	// Parse output like "codex-cli 0.87.0" or "codex-cli 0.120.0-termux"
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return fmt.Errorf("unexpected codex version output: %s", string(out))
	}

	rawVersion := fields[len(fields)-1]
	// Handle "-termux" suffix: extract numeric part for semver comparison
	numericPart := rawVersion
	if idx := strings.Index(rawVersion, "-"); idx > 0 {
		numericPart = rawVersion[:idx]
	}
	version := "v" + numericPart
	minVersion := "v0.81.0"

	if semver.Compare(version, minVersion) < 0 {
		return fmt.Errorf("codex version %s is too old (minimum %s)", rawVersion, "0.81.0")
	}

	return nil
}
