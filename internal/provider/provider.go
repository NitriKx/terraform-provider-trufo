// Package provider implements the Trufo Terraform provider
// (registry.terraform.io/mistral/trufo).
package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	trufo "github.com/mistral/trufo-go"
)

// Env vars, mirroring the trufo-py SDK (TRUFO_API_KEY, TRUFO_ACCESS_TOKEN,
// TRUFO_REFRESH_TOKEN) plus TRUFO_API_URL for the host.
const (
	envAPIURL       = "TRUFO_API_URL"
	envAPIKey       = "TRUFO_API_KEY"
	envAccessToken  = "TRUFO_ACCESS_TOKEN"
	envRefreshToken = "TRUFO_REFRESH_TOKEN"
)

// Regions map to API hosts. api_url overrides region.
const (
	regionGlobal = "global"
	regionEU     = "eu"
	regionTest   = "test"
)

var (
	_ provider.Provider = (*trufoProvider)(nil)
)

// trufoProvider implements the provider.
type trufoProvider struct {
	version string
}

// New returns a provider factory for the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &trufoProvider{version: version}
	}
}

// trufoProviderModel is the provider configuration schema.
type trufoProviderModel struct {
	APIBaseURL   types.String `tfsdk:"api_url"`
	Region       types.String `tfsdk:"region"`
	APIKey       types.String `tfsdk:"api_key"`
	AccessToken  types.String `tfsdk:"access_token"`
	RefreshToken types.String `tfsdk:"refresh_token"`
	AuthFlow     types.String `tfsdk:"auth_flow"`
}

func (p *trufoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "trufo"
	resp.Version = p.version
}

func (p *trufoProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Trufo provider manages resources of the [Trufo](https://trufo.ai) Provenance Platform: " +
			"generator product instances, instance credentials (key enrollment), AI systems and certificates, via the Trufo API. " +
			"All endpoints are authenticated with exactly one credential: an API key (`X-API-Key`) or an access token (`Bearer`).",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "Trufo API base URL, e.g. `https://api.trufo.ai` (production), `https://eu.api.trufo.ai` (EU production) " +
					"or `https://test.api.trufo.ai` (test). Overrides `region`; use it for dedicated hosts. " +
					"Default: `https://api.trufo.ai` (production), or the `" + envAPIURL + "` environment variable.",
				Optional: true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Content region: `global` (`https://api.trufo.ai`, **default**), `eu` (`https://eu.api.trufo.ai`, EU production — media submitted there is processed and stored in EU infrastructure) " +
					"or `test` (`https://test.api.trufo.ai`, test API with test-scoped keys; nothing carries over to production). " +
					"Ignored when `api_url` is set.",
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Trufo API key in `{scope}:{key}` form, sent as `X-API-Key`. " +
					"Default: the `" + envAPIKey + "` environment variable. A key with the `trufo-api` scope is used to run the " +
					"interactive login flow (see `auth_flow`) when no tokens are configured.",
				Optional:  true,
				Sensitive: true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token (Bearer auth), e.g. obtained via the device flow. " +
					"Default: the `" + envAccessToken + "` environment variable. Requires `refresh_token`.",
				Optional:  true,
				Sensitive: true,
			},
			"refresh_token": schema.StringAttribute{
				MarkdownDescription: "Refresh token, rotated on every use; the provider refreshes the access token " +
					"automatically on 401. Default: the `" + envRefreshToken + "` environment variable. Requires `access_token`.",
				Optional:  true,
				Sensitive: true,
			},
			"auth_flow": schema.StringAttribute{
				MarkdownDescription: "How to obtain access/refresh tokens when only a `trufo-api`-scoped `api_key` is set: " +
					"`auto` (loopback browser flow, falling back to the device flow), `loopback` (browser on this machine), " +
					"`device` (prints a verification URL and code) or `none` (use the API key directly). Default: `auto`.",
				Optional: true,
			},
		},
	}
}

func (p *trufoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg trufoProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve with env-var fallbacks.
	apiURL := stringFromEnv(cfg.APIBaseURL, envAPIURL)
	region := stringFromEnv(cfg.Region, "")
	apiKey := stringFromEnv(cfg.APIKey, envAPIKey)
	accessToken := stringFromEnv(cfg.AccessToken, envAccessToken)
	refreshToken := stringFromEnv(cfg.RefreshToken, envRefreshToken)
	authFlow := stringFromEnv(cfg.AuthFlow, "")

	apiURL, err := resolveAPIURL(apiURL, region)
	if err != nil {
		resp.Diagnostics.AddError("Invalid region", err.Error())
		return
	}
	if (accessToken == "") != (refreshToken == "") {
		resp.Diagnostics.AddError(
			"Incomplete token pair",
			fmt.Sprintf("access_token and refresh_token must be set together (check %s and %s).", envAccessToken, envRefreshToken),
		)
		return
	}
	if apiKey == "" && accessToken == "" {
		resp.Diagnostics.AddError(
			"No credentials configured",
			fmt.Sprintf("Set api_key (or %s), or access_token + refresh_token (or %s / %s).", envAPIKey, envAccessToken, envRefreshToken),
		)
		return
	}
	if authFlow == "" {
		authFlow = "auto"
	}
	switch authFlow {
	case "auto", "loopback", "device", "none":
	default:
		resp.Diagnostics.AddError("Invalid auth_flow", fmt.Sprintf("auth_flow must be one of auto, loopback, device, none; got %q", authFlow))
		return
	}

	client := trufo.NewClient(apiURL, trufo.WithAPIKey(apiKey))
	client.SetTokens(accessToken, refreshToken)

	// A trufo-api-scoped key without tokens starts the interactive login flow
	// so that Bearer-only endpoints (gproduct list/info, cert list) work too.
	if accessToken == "" && apiKey != "" && client.APIKeyScope() == "trufo-api" && authFlow != "none" {
		tokens, err := runLoginFlow(ctx, client, authFlow)
		if err != nil {
			resp.Diagnostics.AddError("Trufo login failed", err.Error())
			return
		}
		_ = tokens
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

// runLoginFlow runs the loopback or device flow, printing the verification
// URL when the device flow is used.
func runLoginFlow(ctx context.Context, client *trufo.Client, authFlow string) (*trufo.TokenPair, error) {
	if authFlow == "device" || authFlow == "auto" {
		// Try loopback first for auto; device flow always works headless.
		if authFlow == "auto" {
			if tokens, err := client.LoopbackLogin(ctx, trufo.AppURL, 0); err == nil {
				return tokens, nil
			}
		}
		return client.DeviceFlow(ctx, func(verificationURI, userCode string) {
			fmt.Fprintf(os.Stderr, "\nTrufo sign-in required.\nVisit: %s\nConfirm code: %s\n\n", verificationURI, userCode)
		})
	}
	return client.LoopbackLogin(ctx, trufo.AppURL, 0)
}

func (p *trufoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGeneratorProductDataSource,
		NewCertificatesDataSource,
	}
}

func (p *trufoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewInstanceCredentialResource,
		NewAISystemResource,
	}
}

// resolveAPIURL maps the api_url/region configuration to an API host. With
// neither set, the production global API is the default.
func resolveAPIURL(apiURL, region string) (string, error) {
	if apiURL != "" {
		return apiURL, nil
	}
	switch region {
	case "", regionGlobal:
		return trufo.APIURL, nil
	case regionEU:
		return trufo.APIURLEurope, nil
	case regionTest:
		return trufo.APIURLTest, nil
	default:
		return "", fmt.Errorf("region must be one of %q, %q, %q; got %q", regionGlobal, regionEU, regionTest, region)
	}
}

// stringFromEnv returns the attribute value, or the env var when unset.
func stringFromEnv(attr types.String, envVar string) string {
	if !attr.IsNull() && !attr.IsUnknown() {
		return attr.ValueString()
	}
	return os.Getenv(envVar)
}
