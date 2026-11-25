package trufo

import "context"

// Instance is one signing identity within a generator product: it holds its
// own credentials and is issued its own certificates.
type Instance struct {
	GPIID     string `json:"gpi_id"`
	GPID      string `json:"gp_id,omitempty"`
	OID       string `json:"oid,omitempty"`
	Name      string `json:"name"`
	RecordID  string `json:"record_id,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CreateInstance calls POST /gproduct/instance/create (201) and returns the
// new gpi_id. Requires an active Product Validation on the product.
func (c *Client) CreateInstance(ctx context.Context, gpID, name string) (string, error) {
	var out struct {
		GPIID string `json:"gpi_id"`
	}
	if err := c.post(ctx, GPInstanceCreatePath, map[string]any{"gp_id": gpID, "name": name}, &out); err != nil {
		return "", err
	}
	return out.GPIID, nil
}

// ListInstances calls POST /gproduct/instance/list.
func (c *Client) ListInstances(ctx context.Context, gpID string) ([]Instance, error) {
	var out struct {
		Instances []Instance `json:"instances"`
	}
	if err := c.post(ctx, GPInstanceListPath, map[string]any{"gp_id": gpID}, &out); err != nil {
		return nil, err
	}
	return out.Instances, nil
}

// GetInstance calls POST /gproduct/instance/info.
func (c *Client) GetInstance(ctx context.Context, gpiID string) (*Instance, error) {
	var out Instance
	if err := c.post(ctx, GPInstanceInfoPath, map[string]any{"gpi_id": gpiID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RenameInstance calls POST /gproduct/instance/edit (owner/admin).
func (c *Client) RenameInstance(ctx context.Context, gpiID, name string) error {
	var out statusResponse
	return c.post(ctx, GPInstanceEditPath, map[string]any{"gpi_id": gpiID, "name": name}, &out)
}

// DeleteInstance calls POST /gproduct/instance/delete (owner/admin).
func (c *Client) DeleteInstance(ctx context.Context, gpiID string) error {
	var out statusResponse
	return c.post(ctx, GPInstanceDeletePath, map[string]any{"gpi_id": gpiID}, &out)
}
