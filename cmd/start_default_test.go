//go:build !windows && !darwin

package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/envconfig"
)

// On a non-Termux host the default startApp must not spawn anything: it
// returns an actionable error telling the user to run `ollama serve`.
func TestStartAppNonTermuxReturnsError(t *testing.T) {
	if envconfig.IsTermux() {
		t.Skip("host is Termux; non-Termux default path is not exercised")
	}
	client, err := api.ClientFromEnvironment()
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if err := startApp(context.Background(), client); err == nil {
		t.Fatal("expected error on non-Termux host, got nil")
	}
}

func TestWaitForTermuxServer(t *testing.T) {
	t.Run("returns after heartbeat succeeds", func(t *testing.T) {
		calls := 0
		err := waitForTermuxServer(context.Background(), func(context.Context) error {
			calls++
			if calls < 2 {
				return errors.New("not ready")
			}
			return nil
		}, time.Second, time.Millisecond)
		if err != nil {
			t.Fatalf("waitForTermuxServer: %v", err)
		}
		if calls != 2 {
			t.Fatalf("heartbeat calls = %d, want 2", calls)
		}
	})

	t.Run("fails closed on timeout", func(t *testing.T) {
		err := waitForTermuxServer(context.Background(), func(context.Context) error {
			return errors.New("not ready")
		}, 10*time.Millisecond, time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("expected timeout error, got %v", err)
		}
	})

	t.Run("honors cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := waitForTermuxServer(ctx, func(context.Context) error {
			return errors.New("not ready")
		}, time.Second, time.Millisecond)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	})
}
