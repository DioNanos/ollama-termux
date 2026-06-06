package envconfig

import "testing"

func TestTermuxSystemLinkerExec(t *testing.T) {
	cases := []struct {
		name    string
		termux  string
		knob    string
		context string
		want    bool
	}{
		{"off termux", "", "", "u:r:untrusted_app:s0:c1,c2", false},
		{"force overrides everything", "0.118.0", "force", "", true},
		{"disable overrides everything", "0.118.0", "disable", "u:r:untrusted_app:s0:c1", false},
		{"sideload targetSdk 28 (untrusted_app_27)", "0.118.0", "", "u:r:untrusted_app_27:s0:c150,c256", false},
		{"legacy untrusted_app_25", "0.118.0", "", "u:r:untrusted_app_25:s0:c512", false},
		{"non-app context", "0.118.0", "", "u:r:shell:s0", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERMUX_VERSION", tc.termux)
			t.Setenv("TERMUX_EXEC__SYSTEM_LINKER_EXEC", tc.knob)
			t.Setenv("TERMUX__SE_PROCESS_CONTEXT", tc.context)
			if got := TermuxSystemLinkerExec(); got != tc.want {
				t.Errorf("TermuxSystemLinkerExec() = %v, want %v", got, tc.want)
			}
		})
	}

	// Play Store domain (untrusted_app, targetSdk >= 29) requires the linker,
	// but only when /system/bin/linker64 exists — not on the test host, so it
	// is exercised via the force knob above; here we assert the stat gate.
	t.Run("play store domain without linker64 on host", func(t *testing.T) {
		t.Setenv("TERMUX_VERSION", "0.118.0")
		t.Setenv("TERMUX_EXEC__SYSTEM_LINKER_EXEC", "")
		t.Setenv("TERMUX__SE_PROCESS_CONTEXT", "u:r:untrusted_app:s0:c1,c2")
		want := false // linker64 absent on the build host
		if got := TermuxSystemLinkerExec(); got != want {
			t.Errorf("TermuxSystemLinkerExec() = %v, want %v (host without linker64)", got, want)
		}
	})
}
