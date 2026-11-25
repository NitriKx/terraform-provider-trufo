package trufo

import "context"

// Key algorithms accepted for instance credentials.
const (
	KeyAlgorithmES256 = "ES256" // EC P-256
	KeyAlgorithmEdDSA = "EdDSA" // Ed25519
)

// Credential is a public key registered for an instance; its private key
// signs the client assertion that requests a CSR JWT. Private keys never
// leave the holder's systems. An instance may hold at most two active
// credentials — that cap is what makes key rotation possible.
type Credential struct {
	GPICID       string `json:"gpic_id"`
	GPIID        string `json:"gpi_id"`
	KeyAlgorithm string `json:"key_algorithm"`
	Label        string `json:"label,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	Revoked      bool   `json:"revoked"`
	RevokedAt    string `json:"revoked_at,omitempty"`
}

// RegisterCredential calls POST /gproduct/instance/credential/register (201)
// and returns the new gpic_id. publicKeyPEM is a PEM-encoded public key;
// keyAlgorithm must match the key (ES256 for EC P-256, EdDSA for Ed25519).
func (c *Client) RegisterCredential(ctx context.Context, gpiID, publicKeyPEM, keyAlgorithm, label string) (string, error) {
	body := map[string]any{
		"gpi_id":         gpiID,
		"public_key_pem": publicKeyPEM,
		"key_algorithm":  keyAlgorithm,
	}
	if label != "" {
		body["label"] = label
	}
	var out struct {
		GPICID string `json:"gpic_id"`
	}
	if err := c.post(ctx, GPCredentialRegisterPath, body, &out); err != nil {
		return "", err
	}
	return out.GPICID, nil
}

// ListCredentials calls POST /gproduct/instance/credential/list.
func (c *Client) ListCredentials(ctx context.Context, gpiID string) ([]Credential, error) {
	var out struct {
		Credentials []Credential `json:"credentials"`
	}
	if err := c.post(ctx, GPCredentialListPath, map[string]any{"gpi_id": gpiID}, &out); err != nil {
		return nil, err
	}
	return out.Credentials, nil
}

// EditCredential calls POST /gproduct/instance/credential/edit (owner/admin)
// to change the label.
func (c *Client) EditCredential(ctx context.Context, gpicID, label string) error {
	var out statusResponse
	return c.post(ctx, GPCredentialEditPath, map[string]any{"gpic_id": gpicID, "label": label}, &out)
}

// RevokeCredential calls POST /gproduct/instance/credential/revoke
// (owner/admin). Revocation is permanent.
func (c *Client) RevokeCredential(ctx context.Context, gpicID string) error {
	var out statusResponse
	return c.post(ctx, GPCredentialRevokePath, map[string]any{"gpic_id": gpicID}, &out)
}
