package prompting

import (
	"strings"
	"testing"
)

func TestBuildMessagesYTBot(t *testing.T) {
	messages, err := BuildMessagesYTBot("Great video!", "This is the transcript content.")
	if err != nil {
		t.Fatalf("BuildMessagesYTBot: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	if messages[0].Role != "system" {
		t.Errorf("role = %q, want system", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "Great video!") {
		t.Errorf("content missing comment text")
	}
	if !strings.Contains(messages[0].Content, "This is the transcript content.") {
		t.Errorf("content missing transcript text")
	}
}
