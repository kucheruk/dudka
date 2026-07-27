package agent

import "testing"

func TestValidateAgentNick(t *testing.T) {
	ok, err := FormatAgentNick("Codex", "gpt-5", "mac-mini")
	if err != nil || ok != "Codex·gpt-5·mac-mini" {
		t.Fatalf("format: %q %v", ok, err)
	}
	if err := ValidateAgentNick(ok); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "Codex", "a·b", "a·b·c·d", "·b·c", "a· ·c", "a·b·c "} {
		if ValidateAgentNick(bad) == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
	if LooksLikeAgentNick("Вася") {
		t.Fatal("human must not look like agent")
	}
}
