package launch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditorRunsDoNotRewriteConfig(t *testing.T) {
	tests := []struct {
		name      string
		binary    string
		runner    Runner
		checkPath func(home string) string
	}{
		{
			name:   "kimi",
			binary: "kimi",
			runner: &Kimi{},
			checkPath: func(home string) string {
				return filepath.Join(home, ".kimi", "config.toml")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "pool" && poolsideGOOS == "windows" {
				t.Skip("Poolside is intentionally unsupported on Windows")
			}

			home := t.TempDir()
			setTestHome(t, home)

			binDir := t.TempDir()
			writeFakeBinary(t, binDir, tt.binary)
			if tt.name == "kimi" {
				writeFakeBinary(t, binDir, "curl")
				writeFakeBinary(t, binDir, "bash")
			}
			t.Setenv("PATH", binDir)

			configPath := tt.checkPath(home)
			if err := tt.runner.Run("llama3.2", nil); err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Fatalf("expected Run to leave %s untouched, got err=%v", configPath, err)
			}
		})
	}
}
