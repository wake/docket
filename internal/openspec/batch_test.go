package openspec

import "testing"

func TestUnsafeBatchArg(t *testing.T) {
	for _, bad := range []string{"&", "|", "<", ">", "^", "%", "!", "(", ")", `"`, "'", " ", "é", "add-x&calc"} {
		for _, prog := range []string{`C:\npm\openspec.cmd`, `C:\npm\OPENSPEC.BAT`} {
			if a, ok := unsafeBatchArg(true, []string{prog, "show", bad}); !ok || a != bad {
				t.Errorf("%s with %q: %q %v, want refused", prog, bad, a, ok)
			}
		}
		if _, ok := unsafeBatchArg(true, []string{`C:\npm\openspec.exe`, bad}); ok {
			t.Errorf("an .exe refused %q", bad)
		}
		if _, ok := unsafeBatchArg(false, []string{"/usr/bin/openspec.cmd", bad}); ok {
			t.Errorf("refused %q outside Windows", bad)
		}
	}
	plain := []string{`C:\npm\npx.cmd`, "-y", "@fission-ai/openspec@1.14.0", "show", "add-x", "identity/user-auth",
		"--type", "change", "--deltas-only", "claude,codex", "a_b+c=d:e", ""}
	if a, ok := unsafeBatchArg(true, plain); ok {
		t.Errorf("refused %q", a)
	}
}
