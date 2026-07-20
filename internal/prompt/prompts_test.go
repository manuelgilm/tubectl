package prompt

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

func TestPromptText(t *testing.T) {
	t.Run("with matching tag", func(t *testing.T) {
		m := &RegisteredModel{
			Name: "test-prompt",
			LatestVersions: []ModelVersion{
				{
					Version: "1",
					Tags: []Tag{
						{Key: "mlflow.prompt.text", Value: "Hello {name}"},
						{Key: "other.tag", Value: "irrelevant"},
					},
				},
			},
		}
		if got := m.PromptText(); got != "Hello {name}" {
			t.Errorf("PromptText() = %q, want %q", got, "Hello {name}")
		}
	})

	t.Run("with no matching tag", func(t *testing.T) {
		m := &RegisteredModel{
			Name: "test-prompt",
			LatestVersions: []ModelVersion{
				{
					Version: "1",
					Tags: []Tag{
						{Key: "other.tag", Value: "irrelevant"},
					},
				},
			},
		}
		if got := m.PromptText(); got != "" {
			t.Errorf("PromptText() = %q, want empty", got)
		}
	})

	t.Run("with no latest versions", func(t *testing.T) {
		m := &RegisteredModel{
			Name:           "test-prompt",
			LatestVersions: nil,
		}
		if got := m.PromptText(); got != "" {
			t.Errorf("PromptText() = %q, want empty", got)
		}
	})

}
