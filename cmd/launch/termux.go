package launch

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/format"
	"golang.org/x/mod/semver"
)

// Termux layer for the ollama-termux fork.
//
// Everything Termux-specific in cmd/launch lives in this file (plus
// binresolve.go and codex_vl.go) so upstream files stay as close to upstream
// as possible and future merges remain low-conflict. Behavior is runtime
// gated on envconfig.IsTermux(): off Termux the launcher behaves like
// upstream, with the single addition of the codex-vl integration.

// termuxIntegrationOrder is the launcher menu order on Termux. Only these
// integrations are verified on real Android devices and exposed there.
var termuxIntegrationOrder = []string{"codex", "codex-vl", "qwen", "pi"}

var termuxVisibleIntegrations = func() map[string]bool {
	m := make(map[string]bool, len(termuxIntegrationOrder))
	for _, name := range termuxIntegrationOrder {
		m[name] = true
	}
	return m
}()

func termuxUnsupportedIntegration(name string) error {
	if envconfig.IsTermux() && !termuxVisibleIntegrations[name] {
		return fmt.Errorf("%s is not supported on Termux; available integrations: %s", name, strings.Join(termuxIntegrationOrder, ", "))
	}
	return nil
}

// Supported gates integrations that are not verified on Termux out of the
// interactive launcher and direct launches. Off Termux they all report nil.
// claude-desktop and codex-app already gate themselves upstream by GOOS.
func (c *Claude) Supported() error        { return termuxUnsupportedIntegration("claude") }
func (c *Cline) Supported() error         { return termuxUnsupportedIntegration("cline") }
func (c *Copilot) Supported() error       { return termuxUnsupportedIntegration("copilot") }
func (d *Droid) Supported() error         { return termuxUnsupportedIntegration("droid") }
func (k *Kimi) Supported() error          { return termuxUnsupportedIntegration("kimi") }
func (o *OpenCode) Supported() error      { return termuxUnsupportedIntegration("opencode") }
func (o *Openclaw) Supported() error      { return termuxUnsupportedIntegration("openclaw") }
func (p *Poolside) Supported() error      { return termuxUnsupportedIntegration("pool") }
func (h *Hermes) Supported() error        { return termuxUnsupportedIntegration("hermes") }
func (h *HermesDesktop) Supported() error { return termuxUnsupportedIntegration("hermes-desktop") }
func (v *VSCode) Supported() error        { return termuxUnsupportedIntegration("vscode") }

// termuxRecommendedModels replaces the upstream recommendation list on Termux
// with models sized for smartphone inference plus cloud-backed defaults.
var termuxRecommendedModels = []ModelItem{
	{Name: "qwen3.5:4b", Description: "Recommended local default for coding, reasoning, and visual understanding", Recommended: true, VRAMBytes: 11 * format.GigaByte},
	{Name: "gemma4:e4b", Description: "Recommended local default for reasoning and code generation", Recommended: true, VRAMBytes: 16 * format.GigaByte},
	{Name: "qwen3.5:cloud", Description: "Cloud-backed qwen3.5 with larger context for agentic tool use", Recommended: true, Details: api.ModelDetails{ContextLength: 262_144}, MaxOutputTokens: 32_768},
	{Name: "kimi-k2.6:cloud", Description: "State-of-the-art cloud coding and multimodal agents", Recommended: true, Details: api.ModelDetails{ContextLength: 262_144}, MaxOutputTokens: 262_144},
	{Name: "glm-5.1:cloud", Description: "Cloud reasoning and code generation", Recommended: true, Details: api.ModelDetails{ContextLength: 202_752}, MaxOutputTokens: 131_072},
	{Name: "minimax-m2.7:cloud", Description: "Fast cloud coding and real-world productivity", Recommended: true, Details: api.ModelDetails{ContextLength: 204_800}, MaxOutputTokens: 128_000},
}

func init() {
	registerCodexVLIntegration()

	if envconfig.IsTermux() {
		launcherIntegrationOrder = append([]string{}, termuxIntegrationOrder...)
		recommendedModels = termuxRecommendedModels
		applyTermuxInstallSpecs()
	}

	rebuildIntegrationSpecIndexes()
}

// registerCodexVLIntegration adds the fork-only codex-vl integration to the
// upstream registry. codex-vl is the Vivling-enhanced Codex fork published as
// @mmmbuto/codex-vl on npm.
func registerCodexVLIntegration() {
	integrationSpecs = append(integrationSpecs, &IntegrationSpec{
		Name:        "codex-vl",
		Runner:      &CodexVL{},
		Description: "Vivling-enhanced Codex fork, primary on Termux",
		Install: IntegrationInstallSpec{
			CheckInstalled: func() bool {
				_, err := (&CodexVL{}).findCommand()
				return err == nil
			},
			EnsureInstalled: func() error {
				return termuxEnsureNpmIntegration("Codex VL", "@mmmbuto/codex-vl")
			},
			URL:     "https://www.npmjs.com/package/@mmmbuto/codex-vl",
			Command: []string{"npm", "install", "-g", "@mmmbuto/codex-vl"},
		},
	})
	launcherIntegrationOrder = append(launcherIntegrationOrder, "codex-vl")
}

// applyTermuxInstallSpecs points codex and qwen at the maintained Termux npm
// forks. The upstream install paths either reject Android outright or install
// binaries that do not run under Termux.
func applyTermuxInstallSpecs() {
	for _, spec := range integrationSpecs {
		switch spec.Name {
		case "codex":
			spec.Install = IntegrationInstallSpec{
				CheckInstalled: func() bool {
					_, err := codexTermuxCommand()
					return err == nil
				},
				EnsureInstalled: func() error {
					return termuxEnsureNpmIntegration("Codex (Termux fork)", "@mmmbuto/codex-cli-termux")
				},
				URL:     "https://www.npmjs.com/package/@mmmbuto/codex-cli-termux",
				Command: []string{"npm", "install", "-g", "@mmmbuto/codex-cli-termux"},
			}
		case "qwen":
			spec.Install = IntegrationInstallSpec{
				CheckInstalled: func() bool {
					_, err := qwenTermuxCommand()
					return err == nil
				},
				EnsureInstalled: func() error {
					return termuxEnsureNpmIntegration("Qwen Code (Termux fork)", "@mmmbuto/qwen-code-termux")
				},
				URL:     "https://www.npmjs.com/package/@mmmbuto/qwen-code-termux",
				Command: []string{"npm", "install", "-g", "@mmmbuto/qwen-code-termux"},
			}
		}
	}
}

// termuxEnsureNpmIntegration installs an npm-distributed integration after
// asking for confirmation, mirroring the upstream ensure*Installed flows.
func termuxEnsureNpmIntegration(display, pkg string) error {
	if err := ensureNpmInstalled(); err != nil {
		return err
	}

	ok, err := ConfirmPrompt(display + " is not installed. Install now?")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s installation cancelled", display)
	}

	fmt.Fprintf(os.Stderr, "\nInstalling %s...\n", display)
	cmd := exec.Command("npm", "install", "-g", pkg+"@latest")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", display, err)
	}
	return nil
}

// launchCommandLong returns the `ollama launch` long help. On Termux the
// upstream integration list would be misleading, so it is replaced with the
// verified Termux set.
func launchCommandLong(upstream string) string {
	if !envconfig.IsTermux() {
		return upstream
	}
	return `Launch the Ollama interactive menu, or directly launch a specific integration.

Without arguments, this is equivalent to running 'ollama' directly.
Flags and extra arguments require an integration name.

Supported integrations on Termux:
  codex     Codex (Termux fork)
  codex-vl  Codex VL — Vivling-enhanced fork
  qwen      Qwen Code (Termux fork)
  pi        Pi

Examples:
  ollama launch
  ollama launch codex --model <model>
  ollama launch codex-vl --model <model>
  ollama launch qwen
  ollama launch pi
  ollama launch codex -- -p myprofile (pass extra args to integration)`
}

// codexTermuxCommand resolves the codex binary on Termux, including npm
// global .js entrypoints that need the node interpreter.
func codexTermuxCommand() (resolvedCommand, error) {
	return resolveCommand("codex", termuxPackageEntrypoints("@mmmbuto/codex-cli-termux", "bin/codex.js")...)
}

func qwenTermuxCommand() (resolvedCommand, error) {
	return resolveCommand("qwen", termuxPackageEntrypoints("@mmmbuto/qwen-code-termux", "cli.js")...)
}

// codexExecCommand builds the codex invocation, routing through the Termux
// npm entrypoint resolver when running under Termux.
func codexExecCommand(args ...string) *exec.Cmd {
	if envconfig.IsTermux() {
		if codex, err := codexTermuxCommand(); err == nil {
			return codex.Command(args...)
		}
	}
	return exec.Command("codex", args...)
}

// qwenExecCommand builds the qwen invocation. On Termux it resolves npm .js
// entrypoints through the node interpreter and enables non-interactive
// approvals, matching the other Termux integrations.
func qwenExecCommand(qwenPath string, args []string) *exec.Cmd {
	if envconfig.IsTermux() {
		if !qwenHasFlag(args, "--approval-mode") {
			args = append(args, "--approval-mode", "yolo")
		}
		if spec, err := commandForPath(qwenPath); err == nil {
			return spec.Command(args...)
		}
	}
	return exec.Command(qwenPath, args...)
}

// checkCodexVersionTermux verifies the Termux codex fork version. Fork builds
// report versions like "codex-cli 0.136.1-termux", so the suffix is stripped
// before the semver comparison.
func checkCodexVersionTermux() error {
	codex, err := codexTermuxCommand()
	if err != nil {
		return fmt.Errorf("codex is not installed, install with: npm install -g @mmmbuto/codex-cli-termux")
	}

	out, err := codex.Command("--version").Output()
	if err != nil {
		return fmt.Errorf("failed to get codex version: %w", err)
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return fmt.Errorf("unexpected codex version output: %s", string(out))
	}

	rawVersion := fields[len(fields)-1]
	if err := checkTermuxCodexMinVersion(rawVersion, "v0.134.0"); err != nil {
		return fmt.Errorf("codex %w, update with: npm update -g @mmmbuto/codex-cli-termux", err)
	}
	return nil
}

// checkTermuxCodexMinVersion compares a possibly suffixed fork version
// ("0.136.1-termux", "0.137.0-termux.1") against a minimum semver.
func checkTermuxCodexMinVersion(rawVersion, minVersion string) error {
	numericPart := rawVersion
	if idx := strings.Index(rawVersion, "-"); idx > 0 {
		numericPart = rawVersion[:idx]
	}
	if semver.Compare("v"+numericPart, minVersion) < 0 {
		return fmt.Errorf("version %s is too old, minimum required is %s", rawVersion, strings.TrimPrefix(minVersion, "v"))
	}
	return nil
}
