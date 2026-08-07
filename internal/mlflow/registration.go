package mlflow

import (
	"context"
	"fmt"
	"net/url"
)

type ModelVersionSearchResponse struct {
	ModelVersions []ModelVersion `json:"model_versions"`
	NextPageToken string         `json:"next_page_token,omitempty"`
}

type RegisteredModelResponse struct {
	RegisteredModel RegisteredModel `json:"registered_model"`
}

type RegisteredModel struct {
	Name                 string         `json:"name"`
	CreationTimestamp    int64          `json:"creation_timestamp"`
	LastUpdatedTimestamp int64          `json:"last_updated_timestamp"`
	LatestVersions       []ModelVersion `json:"latest_versions"`
	Tags                 []Tag          `json:"tags"`
}

type ModelVersion struct {
	Name                 string `json:"name"`
	Version              string `json:"version"`
	CreationTimestamp    int64  `json:"creation_timestamp"`
	LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
	CurrentStage         string `json:"current_stage"`
	Description          string `json:"description"`
	Source               string `json:"source"`
	RunID                string `json:"run_id"`
	Status               string `json:"status"`
	Tags                 []Tag  `json:"tags"`
	RunLink              string `json:"run_link"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (m *RegisteredModel) PromptText() string {
	if len(m.LatestVersions) == 0 {
		return ""
	}
	for _, t := range m.LatestVersions[0].Tags {
		if t.Key == "mlflow.prompt.text" {
			return t.Value
		}
	}
	return ""
}

func (c *Client) GetPrompt(ctx context.Context, name string) (*RegisteredModel, error) {
	params := url.Values{}
	params.Set("name", name)

	var resp RegisteredModelResponse
	if err := c.get(ctx, "/api/2.0/mlflow/registered-models/get", params, &resp); err != nil {
		return nil, fmt.Errorf("fetching prompt %q: %w", name, err)
	}

	return &resp.RegisteredModel, nil
}

func (c *Client) ListPrompts(ctx context.Context) ([]ModelVersion, error) {
	params := url.Values{}
	params.Set("filter", `tag.mlflow.prompt.is_prompt = "true"`)

	var resp ModelVersionSearchResponse
	if err := c.get(ctx, "/api/2.0/mlflow/model-versions/search", params, &resp); err != nil {
		return nil, fmt.Errorf("listing prompts: %w", err)
	}
	return resp.ModelVersions, nil
}
