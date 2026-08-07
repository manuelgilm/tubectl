package prompting

import (
	"strings"
	"testing"
)

func TestDefaultBotPromptText(t *testing.T) {
	text := DefaultBotPromptText("Great video!", "Transcript content here.")

	for _, want := range []string{
		"[Automated Reply] Gilsama-Bot",
		"write friendly and helpful replies",
		"Great video!",
		"Transcript content here.",
		"general request",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected prompt to contain %q, got:\n%s", want, text)
		}
	}

	if !strings.Contains(text, "Comment:\nGreat video!") {
		t.Errorf("expected comment after Comment: header, got:\n%s", text)
	}
}
