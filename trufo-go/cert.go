package trufo

import "context"

// Certificate is a listed certificate from POST /cert/list.
type Certificate struct {
	SerialNumber     string `json:"serial_number"`
	LeafType         string `json:"leaf_type"`
	IssueTime        string `json:"issue_time"`
	ExpiryTime       string `json:"expiry_time"`
	RevocationStatus string `json:"revocation_status"`
	RevocationReason string `json:"revocation_reason,omitempty"`
	RevocationTime   string `json:"revocation_time,omitempty"`
	Revocable        bool   `json:"revocable"`
	TrufoManaged     bool   `json:"trufo_managed"`
	GPIID            string `json:"gpi_id,omitempty"`
	GPIName          string `json:"gpi_name,omitempty"`
	GPID             string `json:"gp_id,omitempty"`
	GPName           string `json:"gp_name,omitempty"`
}

// Revocation reasons accepted by POST /cert/revoke.
const (
	RevocationUnspecified          = "unspecified"
	RevocationKeyCompromise        = "key_compromise"
	RevocationAffiliationChanged   = "affiliation_changed"
	RevocationSuperseded           = "superseded"
	RevocationCessationOfOperation = "cessation_of_operation"
)

// ListCertificates calls POST /cert/list.
func (c *Client) ListCertificates(ctx context.Context) ([]Certificate, error) {
	var out struct {
		Certs []Certificate `json:"certs"`
	}
	if err := c.post(ctx, CertListPath, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out.Certs, nil
}

// RevokeCertificate calls POST /cert/revoke (owner/admin, MFA token).
// revocationTime is optional ISO 8601; empty uses the processing time.
// A 502 RevocationFailed means the request did not complete and must not be
// retried blindly — re-list the certificates first.
func (c *Client) RevokeCertificate(ctx context.Context, serialNumber, reason, revocationTime string) error {
	body := map[string]any{
		"serial_number":     serialNumber,
		"revocation_reason": reason,
	}
	if revocationTime != "" {
		body["revocation_time"] = revocationTime
	}
	var out statusResponse
	return c.post(ctx, CertRevokePath, body, &out)
}
