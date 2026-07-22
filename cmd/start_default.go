//go:build !windows && !darwin

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
)

func startApp(ctx context.Context, client *api.Client) error {
	if envconfig.IsTermux() {
		return startTermuxServer(ctx, client)
	}
	return errors.New("could not connect to ollama server, run 'ollama serve' to start it")
}

// startTermuxServer launches `ollama serve` in the background when the
// interactive client (root command or TUI) cannot reach a running server.
//
// Play-Store Termux builds run in an SELinux domain that denies direct
// execve of app-private data files: Go spawns subprocesses with the raw
// syscall and bypasses the termux-exec shim, so the server must be started
// through the Android dynamic linker when TermuxSystemLinkerExec reports it
// is required. Setpgid detaches the server so it survives the client exit.
//
// This preserves the fork's Termux auto-start behaviour after upstream
// consolidated server start into the checkServerHeartbeat -> startApp path
// (upstream removed the former ensureServerRunning helper and its
// background SysProcAttr helpers).
func startTermuxServer(ctx context.Context, client *api.Client) error {
	exe, err := envconfig.TermuxRealExecutable()
	if err != nil {
		return fmt.Errorf("could not find executable: %w", err)
	}

	var serverCmd *exec.Cmd
	if envconfig.TermuxSystemLinkerExec() {
		serverCmd = exec.CommandContext(ctx, envconfig.TermuxSystemLinker, exe, "serve")
	} else {
		serverCmd = exec.CommandContext(ctx, exe, "serve")
	}
	serverCmd.Env = os.Environ()
	serverCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	serverCmd.Stdin = nil
	serverCmd.Stdout = nil
	serverCmd.Stderr = nil
	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	// Detach the background server so it outlives the client process.
	if serverCmd.Process != nil {
		_ = serverCmd.Process.Release()
	}

	return waitForTermuxServer(ctx, client.Heartbeat, 15*time.Second, 500*time.Millisecond)
}

func waitForTermuxServer(ctx context.Context, heartbeat func(context.Context) error, timeout, interval time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("timed out waiting for Termux server to start")
		case <-ticker.C:
			if err := heartbeat(ctx); err == nil {
				return nil
			}
		}
	}
}
