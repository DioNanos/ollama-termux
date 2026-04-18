package launch

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

type resolvedCommand struct {
	Path   string
	Prefix []string
}

func (r resolvedCommand) Command(args ...string) *exec.Cmd {
	if len(r.Prefix) == 0 {
		return exec.Command(r.Path, args...)
	}

	argv := append(append(append([]string{}, r.Prefix...), r.Path), args...)
	return exec.Command(argv[0], argv[1:]...)
}

func resolveCommand(name string, extras ...string) (resolvedCommand, error) {
	candidates := commandCandidates(name, extras...)
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		if _, err := os.Stat(candidate); err != nil {
			continue
		}

		spec, err := commandForPath(candidate)
		if err == nil {
			return spec, nil
		}
	}

	return resolvedCommand{}, exec.ErrNotFound
}

func commandCandidates(name string, extras ...string) []string {
	candidates := make([]string, 0, 1+len(extras)+2)
	candidates = append(candidates, extras...)
	if path, err := exec.LookPath(name); err == nil {
		candidates = append(candidates, path)
	}
	candidates = append(candidates, termuxCommandCandidates(name)...)
	return candidates
}

func termuxCommandCandidates(name string) []string {
	prefixes := []string{
		os.Getenv("PREFIX"),
		"/data/data/com.termux/files/usr",
	}

	candidates := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(prefix, "bin", name))
	}
	return candidates
}

func commandForPath(path string) (resolvedCommand, error) {
	if needsNodeInterpreter(path) {
		nodePath, err := exec.LookPath("node")
		if err != nil {
			return resolvedCommand{}, err
		}
		return resolvedCommand{Path: path, Prefix: []string{nodePath}}, nil
	}
	return resolvedCommand{Path: path}, nil
}

func needsNodeInterpreter(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}

	line := data
	if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
		line = data[:idx]
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return false
	}

	if bytes.HasPrefix(line, []byte("#!")) {
		return bytes.Contains(line, []byte("node"))
	}

	if filepath.Ext(path) == ".js" {
		return true
	}

	if bytes.HasPrefix(line, []byte("import ")) ||
		bytes.HasPrefix(line, []byte("const ")) ||
		bytes.HasPrefix(line, []byte("\"use strict\"")) ||
		bytes.HasPrefix(line, []byte("'use strict'")) {
		return true
	}

	return false
}

func findCommandPath(name string, extras ...string) (string, error) {
	spec, err := resolveCommand(name, extras...)
	if err != nil {
		return "", err
	}
	return spec.Path, nil
}

func termuxOpenURLCommand(url string) (*exec.Cmd, error) {
	spec, err := resolveCommand("termux-open-url")
	if err != nil {
		return nil, err
	}
	return spec.Command(url), nil
}

func termuxPackageEntrypoint(pkg, relativePath string) string {
	prefixes := []string{
		os.Getenv("PREFIX"),
		"/data/data/com.termux/files/usr",
	}

	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		return filepath.Join(prefix, "lib", "node_modules", pkg, filepath.FromSlash(relativePath))
	}

	return ""
}
