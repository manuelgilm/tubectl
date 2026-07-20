package prompt

type ModelVersionSearchResponse struct {
    ModelVersions []ModelVersion `json:"model_versions"`
    NextPageToken string         `json:"next_page_token,omitempty"`
}

type RegisteredModelResponse struct {
	RegisteredModel RegisteredModel 	`json:"registered_model"`
}

type RegisteredModel struct {
	Name					string			`json:"name"`
	CreationTimestamp		int64			`json:"creation_timestamp"`
	LastUpdatedTimestamp	int64			`json:"last_updated_timestamp"`
	LatestVersions      	[]ModelVersion 	`json:"latest_versions"`
	Tags					[]Tag			`json:"tags"`
}


type ModelVersion struct {
    Name                string `json:"name"`
    Version             string `json:"version"`
    CreationTimestamp    int64  `json:"creation_timestamp"`
    LastUpdatedTimestamp int64  `json:"last_updated_timestamp"`
    CurrentStage        string `json:"current_stage"`
    Description         string `json:"description"`
    Source              string `json:"source"`
    RunID               string `json:"run_id"`
    Status              string `json:"status"`
    Tags                []Tag  `json:"tags"`
    RunLink             string `json:"run_link"`
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