package launch

import (
	"fmt"
	"os"
	"strings"

	"github.com/ollama/ollama/envconfig"
)

// Qwen implements Runner for Qwen Code integration.
type Qwen struct{}

func (q *Qwen) String() string { return "Qwen Code" }

func (q *Qwen) baseURL() string {
	return strings.TrimSuffix(envconfig.Host().String(), "/") + "/v1"
}

func (q *Qwen) args(model string, extra []string) []string {
	args := []string{
		"--auth-type", "openai",
		"--openai-api-key", "ollama",
		"--openai-base-url", q.baseURL(),
		"--approval-mode", "yolo",
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extra...)
	return args
}

func (q *Qwen) findCommand() (resolvedCommand, error) {
	termuxPackageCLI := termuxPackageEntrypoint("@mmmbuto/qwen-code-termux", "cli.js")
	return resolveCommand("qwen", termuxPackageCLI)
}

func (q *Qwen) findPath() (string, error) {
	termuxPackageCLI := termuxPackageEntrypoint("@mmmbuto/qwen-code-termux", "cli.js")
	return findCommandPath("qwen", termuxPackageCLI)
}

func (q *Qwen) Run(model string, args []string) error {
	qwen, err := q.findCommand()
	if err != nil {
		return fmt.Errorf("qwen is not installed\n\nInstall with:\n  npm install -g @mmmbuto/qwen-code-termux  (recommended on Termux)\n  or\n  npm install -g @qwen-code/qwen-code")
	}

	cmd := qwen.Command(q.args(model, args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=ollama",
		"OPENAI_BASE_URL="+q.baseURL(),
	)
	return cmd.Run()
}
