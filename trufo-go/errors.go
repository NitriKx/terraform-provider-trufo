package trufo

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FieldError is one entry of a 422 validation error, where detail is a list
// of {type, loc, msg} objects (FastAPI style).
type FieldError struct {
	Type string   `json:"type"`
	Loc  []string `json:"loc"`
	Msg  string   `json:"msg"`
}

// APIError is the error returned for any non-2xx Trufo API response. Trufo
// errors carry a stable machine-readable code in the JSON body's detail field
// (e.g. {"detail": "InsufficientPermissionForAction"}); 422 responses instead
// carry detail as a list of field errors.
type APIError struct {
	StatusCode int
	// Detail is the stable error code, or the raw body when it is not JSON.
	Detail string
	// Action is set for InsufficientPermissionForAction, which names the
	// missing permission.
	Action string
	// FieldErrors is set for 422 validation failures.
	FieldErrors []FieldError
	// RequestID is the X-Request-ID response header, to quote when
	// contacting support.
	RequestID string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "trufo API error (HTTP %d)", e.StatusCode)
	if e.Detail != "" {
		fmt.Fprintf(&b, ": %s", e.Detail)
	}
	if e.Action != "" {
		fmt.Fprintf(&b, " (action: %s)", e.Action)
	}
	if len(e.FieldErrors) > 0 {
		parts := make([]string, 0, len(e.FieldErrors))
		for _, fe := range e.FieldErrors {
			parts = append(parts, fmt.Sprintf("%s: %s", strings.Join(fe.Loc, "."), fe.Msg))
		}
		fmt.Fprintf(&b, ": %s", strings.Join(parts, "; "))
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " [request-id: %s]", e.RequestID)
	}
	return b.String()
}

// Is reports whether err is an *APIError with the given status code and, if
// detail is non-empty, that error code.
func Is(err error, statusCode int, detail string) bool {
	ae, ok := err.(*APIError)
	if !ok || ae.StatusCode != statusCode {
		return false
	}
	return detail == "" || ae.Detail == detail
}

// errorBody is the JSON error envelope.
type errorBody struct {
	Detail json.RawMessage `json:"detail"`
	Action string          `json:"action"`
}

// parseError builds an *APIError from a non-2xx response.
func parseError(statusCode int, requestID string, body []byte) *APIError {
	ae := &APIError{StatusCode: statusCode, RequestID: requestID}
	var eb errorBody
	if err := json.Unmarshal(body, &eb); err == nil && len(eb.Detail) > 0 {
		// detail is a string for most errors, a list for 422.
		var s string
		if err := json.Unmarshal(eb.Detail, &s); err == nil {
			ae.Detail = s
			ae.Action = eb.Action
			return ae
		}
		var fes []FieldError
		if err := json.Unmarshal(eb.Detail, &fes); err == nil {
			ae.FieldErrors = fes
			return ae
		}
		ae.Detail = string(eb.Detail)
		return ae
	}
	if len(body) > 0 {
		ae.Detail = string(body)
	}
	return ae
}
