package llm

import (
	"slices"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/format"
	"github.com/ollama/ollama/ml"
)

func TestTermuxRunnerLibraryPaths(t *testing.T) {
	t.Run("off Termux", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "")
		t.Setenv("PREFIX", "/tmp/not-termux")
		if got := termuxRunnerLibraryPaths(); got != nil {
			t.Fatalf("termuxRunnerLibraryPaths() = %v, want nil", got)
		}
	})

	t.Run("system Vulkan loader precedes Termux libraries", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "0.118.0")
		t.Setenv("PREFIX", "/custom/termux/usr")
		want := []string{
			"/system/lib64",
			"/custom/termux/usr/lib",
			"/data/data/com.termux/files/usr/lib",
		}
		if got := termuxRunnerLibraryPaths(); !slices.Equal(got, want) {
			t.Fatalf("termuxRunnerLibraryPaths() = %v, want %v", got, want)
		}
	})

	t.Run("default prefix is not duplicated", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "0.118.0")
		t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
		want := []string{"/system/lib64", "/data/data/com.termux/files/usr/lib"}
		if got := termuxRunnerLibraryPaths(); !slices.Equal(got, want) {
			t.Fatalf("termuxRunnerLibraryPaths() = %v, want %v", got, want)
		}
	})
}

func TestLimitTermuxContextUsesConservativeBudget(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")
	tests := []struct {
		name string
		free uint64
		want int
	}{
		{name: "under 2 GiB", free: format.GibiByte, want: 2048},
		{name: "under 4 GiB", free: 3 * format.GibiByte, want: 4096},
		{name: "under 8 GiB", free: 6 * format.GibiByte, want: 8192},
		{name: "under 12 GiB", free: 10 * format.GibiByte, want: 16384},
		{name: "at least 12 GiB", free: 12 * format.GibiByte, want: 32768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := api.Options{Runner: api.Runner{NumCtx: 262144}}
			limitTermuxContext(&opts, ml.SystemInfo{FreeMemory: tt.free})
			if opts.NumCtx != tt.want {
				t.Fatalf("NumCtx = %d, want %d", opts.NumCtx, tt.want)
			}
		})
	}
}
