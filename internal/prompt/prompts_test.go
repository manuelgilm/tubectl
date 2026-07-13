package prompt

import "testing"

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
