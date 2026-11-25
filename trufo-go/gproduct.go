package trufo

import "context"

// GProduct is a generator product: one C2PA-producing product. Products are
// created in the Trufo dashboard (as part of a Product Validation
// application); the API can list, inspect, rename and cancel them.
type GProduct struct {
	GPID               string `json:"gp_id"`
	OID                string `json:"oid,omitempty"`
	Name               string `json:"name"`
	TP                 string `json:"tp,omitempty"`
	Implementation     string `json:"implementation,omitempty"`
	PVStatus           string `json:"pv_status,omitempty"`
	PVProductName      string `json:"pv_product_name,omitempty"`
	PVAssuranceLevel   string `json:"pv_assurance_level,omitempty"`
	PVActive           bool   `json:"pv_active,omitempty"`
	Branded            bool   `json:"branded,omitempty"`
	BPVStatus          string `json:"bpv_status,omitempty"`
	ValidatedExpiresAt string `json:"validated_expires_at,omitempty"`
	CancelAt           string `json:"cancel_at,omitempty"`
	CreatedBy          string `json:"created_by,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
}

// statusResponse is the generic {"status": "..."} write response.
type statusResponse struct {
	Status string `json:"status"`
}

// ListGProducts calls POST /gproduct/list. The oid field is required by the
// request schema; the server uses the credential's organization.
func (c *Client) ListGProducts(ctx context.Context, oid string) ([]GProduct, error) {
	var out struct {
		GProducts []GProduct `json:"gproducts"`
	}
	if err := c.post(ctx, GProductListPath, map[string]any{"oid": oid}, &out); err != nil {
		return nil, err
	}
	return out.GProducts, nil
}

// GetGProduct calls POST /gproduct/info.
func (c *Client) GetGProduct(ctx context.Context, gpID string) (*GProduct, error) {
	var out GProduct
	if err := c.post(ctx, GProductInfoPath, map[string]any{"gp_id": gpID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RenameGProduct calls POST /gproduct/info/edit (owner/admin).
func (c *Client) RenameGProduct(ctx context.Context, gpID, name string) error {
	var out statusResponse
	return c.post(ctx, GProductInfoEditPath, map[string]any{"gp_id": gpID, "name": name}, &out)
}

// CancelGProduct calls POST /gproduct/cancel (owner/admin).
func (c *Client) CancelGProduct(ctx context.Context, gpID string) error {
	var out statusResponse
	return c.post(ctx, GProductCancelPath, map[string]any{"gp_id": gpID}, &out)
}
