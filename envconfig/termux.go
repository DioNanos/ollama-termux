package envconfig

import (
	"os"
	"path/filepath"
	"strings"
)

// TermuxSystemLinker is the absolute path of the Android dynamic linker used
// to execute app-data binaries on SELinux domains that deny direct execve.
const TermuxSystemLinker = "/system/bin/linker64"

// TermuxSystemLinkerExec reports whether subprocesses must be started through
// the system linker on this Termux install.
//
// Termux builds targeting SDK >= 29 (Google Play) run in an SELinux domain
// that denies execve of app data files. Shell processes still work because
// termux-exec rewrites libc exec calls into "/system/bin/linker64 <path>",
// but Go spawns subprocesses with the raw syscall and bypasses that shim, so
// the routing has to be done explicitly. F-Droid/sideload builds (targetSdk
// 28, untrusted_app_25/27 domains) keep direct exec.
//
// TERMUX_EXEC__SYSTEM_LINKER_EXEC=force|disable overrides the detection,
// mirroring the termux-exec knob.
func TermuxSystemLinkerExec() bool {
	if !IsTermux() {
		return false
	}

	switch strings.ToLower(os.Getenv("TERMUX_EXEC__SYSTEM_LINKER_EXEC")) {
	case "disable":
		return false
	case "force":
		return true
	}

	context := os.Getenv("TERMUX__SE_PROCESS_CONTEXT")
	if context == "" {
		if b, err := os.ReadFile("/proc/self/attr/current"); err == nil {
			context = string(b)
		}
	}
	context = strings.TrimRight(context, "\x00\n")

	if !strings.Contains(context, ":untrusted_app") {
		return false
	}
	// untrusted_app_25 / untrusted_app_27 (targetSdk <= 28) may exec app data
	// files directly; every later domain is denied by SELinux.
	if strings.Contains(context, ":untrusted_app_25:") || strings.Contains(context, ":untrusted_app_27:") {
		return false
	}
	if _, err := os.Stat(TermuxSystemLinker); err != nil {
		return false
	}
	return true
}

// TermuxRealExecutable returns the path of the current executable. When the
// process was started through the system linker, /proc/self/exe (and thus
// os.Executable) points at linker64 instead of the real binary; termux-exec
// publishes the real path in TERMUX_EXEC__PROC_SELF_EXE, with argv[0] as the
// fallback (the linker rewrites it to the loaded program).
func TermuxRealExecutable() (string, error) {
	exe, err := os.Executable()

	if IsTermux() {
		if p := os.Getenv("TERMUX_EXEC__PROC_SELF_EXE"); p != "" {
			return p, nil
		}
		if err == nil && exe == TermuxSystemLinker && len(os.Args) > 0 {
			if abs, absErr := filepath.Abs(os.Args[0]); absErr == nil {
				return abs, nil
			}
		}
	}

	return exe, err
}
