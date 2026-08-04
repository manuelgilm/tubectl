package mlflow

import "context"

// GatewayEndpoint summarizes an MLflow gateway endpoint for diagnostics.
// Field names match the server proto JSON (camelCase).
type GatewayEndpoint struct {
	EndpointID      string `json:"endpointId"`
	Name            string `json:"name"`
	RoutingStrategy string `json:"routingStrategy"`
	UsageTracking   bool   `json:"usageTracking"`
	ExperimentID    string `json:"experimentId"`
}

// ListGatewayEndpoints returns all gateway endpoints configured on the server.
func (c *Client) ListGatewayEndpoints(ctx context.Context) ([]GatewayEndpoint, error) {
	var resp struct {
		Endpoints []GatewayEndpoint `json:"endpoints"`
	}
	if err := c.get(ctx, "/api/2.0/mlflow/gateway/endpoints/list", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Endpoints, nil
}
