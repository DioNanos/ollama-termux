//go:build !windows && !darwin

package cmd

import (
	"context"
	"testing"

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
