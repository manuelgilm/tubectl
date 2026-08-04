package mlflow

import (
	"context"
	"net/url"
)

// Experiment is minimal metadata for an MLflow experiment.
type Experiment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListExperiments returns all active experiments on the server.
func (c *Client) ListExperiments(ctx context.Context) ([]Experiment, error) {
	params := url.Values{}
	params.Set("view_type", "ACTIVE_ONLY")

	var resp struct {
		Experiments []Experiment `json:"experiments"`
	}
	if err := c.get(ctx, "/api/2.0/mlflow/experiments/search", params, &resp); err != nil {
		return nil, err
	}
	return resp.Experiments, nil
}

// ExperimentIDByName resolves an experiment name to its ID, returning "" if
// no active experiment with that name exists.
func (c *Client) ExperimentIDByName(ctx context.Context, name string) (string, error) {
	experiments, err := c.ListExperiments(ctx)
	if err != nil {
		return "", err
	}
	for _, e := range experiments {
		if e.Name == name {
			return e.ID, nil
		}
	}
	return "", nil
}
