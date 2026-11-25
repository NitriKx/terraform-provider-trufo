package trufo

import "context"

// AI system involvement modes.
const (
	AIInvolvementGenerated = "ai_generated"
	AIInvolvementModified  = "ai_modified"
)

// AI system statuses.
const (
	AISystemStatusActive   = "active"
	AISystemStatusInactive = "inactive"
)

// AISystem is a registered AI system (POST /ai-system/add). Its details
// cannot change after registration: register each new version as a new AI
// system. Production hosts only.
type AISystem struct {
	AISystemID               string         `json:"ai_system_id"`
	Name                     string         `json:"name"`
	Version                  string         `json:"version"`
	AIInvolvementModes       []string       `json:"ai_involvement_modes"`
	C2PAFirstPartyAttestable bool           `json:"c2pa_first_party_attestable"`
	C2PAAIDisclosure         map[string]any `json:"c2pa_ai_disclosure,omitempty"`
	Nickname                 string         `json:"nickname,omitempty"`
	Status                   string         `json:"status,omitempty"`
}

// AddAISystem calls POST /ai-system/add (201) and returns the ai_system_id.
// c2paFirstPartyAttestable requires the Created Assertion Acknowledgement to
// have been accepted in the dashboard first.
func (c *Client) AddAISystem(ctx context.Context, sys AISystem) (string, error) {
	body := map[string]any{
		"name":                 sys.Name,
		"version":              sys.Version,
		"ai_involvement_modes": sys.AIInvolvementModes,
	}
	if sys.C2PAFirstPartyAttestable {
		body["c2pa_first_party_attestable"] = true
	}
	if sys.C2PAAIDisclosure != nil {
		body["c2pa_ai_disclosure"] = sys.C2PAAIDisclosure
	}
	if sys.Nickname != "" {
		body["nickname"] = sys.Nickname
	}
	var out struct {
		AISystemID string `json:"ai_system_id"`
	}
	if err := c.post(ctx, AISystemAddPath, body, &out); err != nil {
		return "", err
	}
	return out.AISystemID, nil
}

// ListAISystems calls POST /ai-system/list (up to 1000 items, newest first).
func (c *Client) ListAISystems(ctx context.Context) ([]AISystem, error) {
	var out struct {
		Items []AISystem `json:"items"`
	}
	if err := c.post(ctx, AISystemListPath, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// SetAISystemStatus calls POST /ai-system/status. Idempotent; nothing is
// deleted — inactive rejects new signs and embeds using the AI system.
func (c *Client) SetAISystemStatus(ctx context.Context, aiSystemID, status string) (*AISystem, error) {
	var out AISystem
	if err := c.post(ctx, AISystemStatusPath, map[string]any{"ai_system_id": aiSystemID, "status": status}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
