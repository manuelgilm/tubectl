package mlflow

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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
	return m.LatestVersions[0].PromptText()
}

// PromptText returns the prompt text carried by this model version, or "" if
// the version has no mlflow.prompt.text tag.
func (mv *ModelVersion) PromptText() string {
	for _, t := range mv.Tags {
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

// GetPromptVersionByAlias resolves the model version an alias points to and
// returns it as a RegisteredModel carrying that single version, so callers can
// read the prompt text with PromptText().
func (c *Client) GetPromptVersionByAlias(ctx context.Context, name, alias string) (*RegisteredModel, error) {
	params := url.Values{}
	params.Set("name", name)
	params.Set("alias", alias)

	var resp struct {
		ModelVersion ModelVersion `json:"model_version"`
	}
	if err := c.get(ctx, "/api/2.0/mlflow/registered-models/alias", params, &resp); err != nil {
		return nil, fmt.Errorf("fetching prompt %q by alias %q: %w", name, alias, err)
	}

	return &RegisteredModel{
		Name:           name,
		LatestVersions: []ModelVersion{resp.ModelVersion},
	}, nil
}

// GetPromptRef resolves a prompt reference using the MLflow-style "name@alias"
// syntax. When an alias is present the prompt is fetched through the model
// registry alias endpoint; otherwise the latest version of the named prompt is
// returned.
func (c *Client) GetPromptRef(ctx context.Context, ref string) (*RegisteredModel, error) {
	name, alias := splitPromptRef(ref)
	if name == "" {
		return nil, fmt.Errorf("empty prompt name in reference %q", ref)
	}
	if alias == "" {
		return c.GetPrompt(ctx, name)
	}
	return c.GetPromptVersionByAlias(ctx, name, alias)
}

// splitPromptRef splits "name@alias" on the last '@'. An empty alias part
// ("name@") is treated as no alias.
func splitPromptRef(ref string) (name, alias string) {
	idx := strings.LastIndex(ref, "@")
	if idx < 0 {
		return ref, ""
	}
	return ref[:idx], ref[idx+1:]
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
