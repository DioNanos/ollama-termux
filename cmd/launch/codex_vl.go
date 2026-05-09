package launch

import (
	"fmt"
	"os"
	"strings"

	"github.com/ollama/ollama/envconfig"
	"golang.org/x/mod/semver"
)

// CodexVL implements Runner for Codex VL integration.
// Codex VL is the Vivling-enhanced fork of Codex maintained at git@forge:dag/codex-vl.git,
// published as @mmmbuto/codex-vl on npm.
type CodexVL struct{}

func (c *CodexVL) String() string { return "Codex VL" }

const codexVLProfileName = "ollama-launch"

func (c *CodexVL) args(model string, extra []string) []string {
	args := []string{"--profile", codexVLProfileName, "--dangerously-bypass-approvals-and-sandbox"}
	if model != "" {
		args = append(args, "-m", model)
	}
	args = append(args, extra...)
	return args
}

func (c *CodexVL) Run(model string, args []string) error {
	codexVL, err := c.findCommand()
	if err != nil {
		return fmt.Errorf("codex-vl is not installed\n\nInstall with:\n  npm install -g @mmmbuto/codex-vl  (recommended on Termux)")
	}

	if err := checkCodexVLVersion(codexVL); err != nil {
		return err
	}

	if err := ensureCodexVLConfig(); err != nil {
		return fmt.Errorf("failed to configure codex-vl: %w", err)
	}

	cmd := codexVL.Command(c.args(model, args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=ollama",
	)
	return cmd.Run()
}

func (c *CodexVL) findCommand() (resolvedCommand, error) {
	termuxPackageCLI := termuxPackageEntrypoint("@mmmbuto/codex-vl", "bin/codex.js")
	return resolveCommand("codex-vl", termuxPackageCLI)
}

func (c *CodexVL) findPath() (string, error) {
	termuxPackageCLI := termuxPackageEntrypoint("@mmmbuto/codex-vl", "bin/codex.js")
	return findCommandPath("codex-vl", termuxPackageCLI)
}

// ensureCodexVLConfig writes the ollama-launch profile to ~/.codex/config.toml.
// Codex VL shares the same config format as upstream Codex.
func ensureCodexVLConfig() error {
	return ensureCodexConfig()
}

func checkCodexVLVersion(codexVL resolvedCommand) error {
	out, err := codexVL.Command("--version").Output()
	if err != nil {
		return fmt.Errorf("failed to get codex-vl version: %w", err)
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return fmt.Errorf("unexpected codex-vl version output: %s", string(out))
	}

	rawVersion := fields[len(fields)-1]
	numericPart := rawVersion
	if idx := strings.Index(rawVersion, "-"); idx > 0 {
		numericPart = rawVersion[:idx]
	}
	version := "v" + numericPart
	minVersion := "v0.130.0"

	if semver.Compare(version, minVersion) < 0 {
		return fmt.Errorf("codex-vl version %s is too old (minimum %s)", rawVersion, "0.130.0")
	}

	return nil
}
