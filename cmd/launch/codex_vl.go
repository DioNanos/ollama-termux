package launch

import (
	"fmt"
	"os"
	"strings"

	"github.com/ollama/ollama/envconfig"
)

// CodexVL implements Runner for the Codex VL integration.
// Codex VL is the Vivling-enhanced fork of Codex published as
// @mmmbuto/codex-vl on npm (https://github.com/DioNanos/codex-vl).
// It shares the upstream Codex config format, so the ollama-launch profile
// and model catalog written by ensureCodexConfig are reused as-is.
type CodexVL struct{}

func (c *CodexVL) String() string { return "Codex VL" }

func (c *CodexVL) args(model, modelCatalogPath string, extra []string) ([]string, error) {
	if err := codexValidateExtraArgs(extra); err != nil {
		return nil, err
	}

	args := []string{"--profile", codexProfileName}
	for _, override := range codexManagedConfigOverrides(modelCatalogPath) {
		args = append(args, "-c", override)
	}
	if envconfig.IsTermux() {
		// Termux runs headless/autonomous; per-action approval prompts and
		// the sandbox are unavailable on Android.
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}
	if model != "" {
		args = append(args, "-m", model)
	}
	args = append(args, extra...)
	return args, nil
}

func (c *CodexVL) Run(model string, models []LaunchModel, args []string) error {
	codexVL, err := c.findCommand()
	if err != nil {
		return fmt.Errorf("codex-vl is not installed\n\nInstall with:\n  npm install -g @mmmbuto/codex-vl")
	}

	if err := checkCodexVLVersion(codexVL); err != nil {
		return err
	}

	if err := ensureCodexConfig(model, models); err != nil {
		return fmt.Errorf("failed to configure codex-vl: %w", err)
	}

	catalogPath, err := codexModelCatalogPath()
	if err != nil {
		return fmt.Errorf("failed to configure codex-vl: %w", err)
	}

	codexArgs, err := c.args(model, catalogPath, args)
	if err != nil {
		return fmt.Errorf("failed to configure codex-vl: %w", err)
	}

	cmd := codexVL.Command(codexArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=ollama",
	)
	return cmd.Run()
}

func (c *CodexVL) findCommand() (resolvedCommand, error) {
	return resolveCommand("codex-vl", termuxPackageEntrypoints("@mmmbuto/codex-vl", "bin/codex.js")...)
}

func (c *CodexVL) findPath() (string, error) {
	return findCommandPath("codex-vl", termuxPackageEntrypoints("@mmmbuto/codex-vl", "bin/codex.js")...)
}

// checkCodexVLVersion verifies the installed codex-vl supports the managed
// profile and model catalog flow (upstream Codex >= 0.134.0 baseline).
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
	if err := checkTermuxCodexMinVersion(rawVersion, "v0.134.0"); err != nil {
		return fmt.Errorf("codex-vl %w, update with: npm update -g @mmmbuto/codex-vl", err)
	}
	return nil
}
