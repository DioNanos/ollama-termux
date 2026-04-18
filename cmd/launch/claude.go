package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/ollama/ollama/envconfig"
)

// Claude implements Runner for Claude Code integration.
type Claude struct{}

func (c *Claude) String() string { return "Claude Code" }

func (c *Claude) args(model string, extra []string) []string {
	var args []string
	args = append(args, "--dangerously-skip-permissions")
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extra...)
	return args
}

func (c *Claude) findPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	name := "claude"
	if runtime.GOOS == "windows" {
		name = "claude.exe"
	}
	fallback := filepath.Join(home, ".claude", "local", name)
	termuxPackageCLI := termuxPackageEntrypoint("@anthropic-ai/claude-code", "cli.js")
	return findCommandPath("claude", fallback, termuxPackageCLI)
}

func (c *Claude) findCommand() (resolvedCommand, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return resolvedCommand{}, err
	}
	name := "claude"
	if runtime.GOOS == "windows" {
		name = "claude.exe"
	}
	fallback := filepath.Join(home, ".claude", "local", name)
	termuxPackageCLI := termuxPackageEntrypoint("@anthropic-ai/claude-code", "cli.js")
	return resolveCommand("claude", fallback, termuxPackageCLI)
}

func (c *Claude) Run(model string, args []string) error {
	claude, err := c.findCommand()
	if err != nil {
		return fmt.Errorf("claude is not installed\n\nOn Termux install:\n  npm install -g @anthropic-ai/claude-code@2.1.112\n\nNote: @anthropic-ai/claude-code 2.1.113 and newer no longer ship native Termux support")
	}

	cmd := claude.Command(c.args(model, args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := append(os.Environ(),
		"ANTHROPIC_BASE_URL="+envconfig.Host().String(),
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_AUTH_TOKEN=ollama",
		"CLAUDE_CODE_ATTRIBUTION_HEADER=0",
	)

	env = append(env, c.modelEnvVars(model)...)

	cmd.Env = env
	return cmd.Run()
}

// modelEnvVars returns Claude Code env vars that route all model tiers through Ollama.
func (c *Claude) modelEnvVars(model string) []string {
	env := []string{
		"ANTHROPIC_DEFAULT_OPUS_MODEL=" + model,
		"ANTHROPIC_DEFAULT_SONNET_MODEL=" + model,
		"ANTHROPIC_DEFAULT_HAIKU_MODEL=" + model,
		"CLAUDE_CODE_SUBAGENT_MODEL=" + model,
	}

	if isCloudModelName(model) {
		if l, ok := lookupCloudModelLimit(model); ok {
			env = append(env, "CLAUDE_CODE_AUTO_COMPACT_WINDOW="+strconv.Itoa(l.Context))
		}
	}

	return env
}
