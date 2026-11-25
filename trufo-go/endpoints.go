// Package trufo provides a Go client for the Trufo API (api.trufo.ai).
//
// It is a port of the Python SDK github.com/trufo-ai/trufo-py, covering the
// account/device auth flows and the Registration Authority resources
// (generator products, instances, instance credentials, certificates) plus
// AI systems. Media-processing endpoints (C2PA signing, watermarking, EST
// enrollment) are intentionally out of scope.
package trufo

import "os"

// API hosts.
const (
	APIURL       = "https://api.trufo.ai"
	APIURLEurope = "https://eu.api.trufo.ai"
	APIURLTest   = "https://test.api.trufo.ai"
	CAURL        = "https://ca.trufo.ai"
	CAURLTest    = "https://test.ca.trufo.ai"
	OCSPURL      = "https://ocsp.trufo.ai"
	TSAURL       = "https://tsa.trufo.ai"
)

// AppURL is the webapp origin the loopback login sends the browser to. Like
// trufo-py's TRUFO_APP_URL, it can be overridden via the environment.
var AppURL = func() string {
	if v := os.Getenv("TRUFO_APP_URL"); v != "" {
		return v
	}
	return "https://app.trufo.ai"
}()

// Account / auth endpoints (RFC 8628 device flow, RFC 8252 loopback flow).
const (
	DeviceAuthorizePath = "/account/device/authorize"
	DeviceTokenPath     = "/account/device/token"
	LoopbackTokenPath   = "/account/loopback/token"
	LoopbackAuthPath    = "/loopback" // path on AppURL, not on the API
	AccountRefreshPath  = "/account/refresh"
	AccountLogoutPath   = "/account/logout"
)

// Registration Authority endpoints.
const (
	RACSRJWTPath         = "/ra/c2pa/csr-jwt"
	RACAWGInterimCSRPath = "/ra/cawg-interim/csr-jwt"
)

// Generator product endpoints. Products are created in the Trufo dashboard;
// the API can list, inspect, rename and cancel them.
const (
	GProductListPath     = "/gproduct/list"
	GProductInfoPath     = "/gproduct/info"
	GProductInfoEditPath = "/gproduct/info/edit"
	GProductCancelPath   = "/gproduct/cancel"
)

// Generator product instance endpoints.
const (
	GPInstanceCreatePath = "/gproduct/instance/create"
	GPInstanceListPath   = "/gproduct/instance/list"
	GPInstanceInfoPath   = "/gproduct/instance/info"
	GPInstanceEditPath   = "/gproduct/instance/edit"
	GPInstanceDeletePath = "/gproduct/instance/delete"
)

// Instance credential endpoints ("instance key enrollment").
const (
	GPCredentialRegisterPath = "/gproduct/instance/credential/register"
	GPCredentialListPath     = "/gproduct/instance/credential/list"
	GPCredentialEditPath     = "/gproduct/instance/credential/edit"
	GPCredentialRevokePath   = "/gproduct/instance/credential/revoke"
)

// Certificate endpoints (Registration Authority side; OCSP/TSA are separate
// services and out of scope here).
const (
	CertListPath   = "/cert/list"
	CertRevokePath = "/cert/revoke"
)

// AI system endpoints (production hosts only).
const (
	AISystemAddPath    = "/ai-system/add"
	AISystemListPath   = "/ai-system/list"
	AISystemStatusPath = "/ai-system/status"
)

// Tasks (the only GET endpoint in the API).
const (
	TasksPath = "/tasks/"
)
