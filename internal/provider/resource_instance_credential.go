package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	trufo "github.com/NitriKx/terraform-trufo-ai-provider/trufo-go"
)

var (
	_ resource.Resource                = (*instanceCredentialResource)(nil)
	_ resource.ResourceWithImportState = (*instanceCredentialResource)(nil)
)

// NewInstanceCredentialResource creates a trufo_instance_credential resource.
func NewInstanceCredentialResource() resource.Resource { return &instanceCredentialResource{} }

type instanceCredentialResource struct {
	client *trufo.Client
}

func (r *instanceCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*trufo.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected *trufo.Client.")
		return
	}
	r.client = c
}

type instanceCredentialResourceModel struct {
	ID           types.String `tfsdk:"id"`
	GPIID        types.String `tfsdk:"gpi_id"`
	PublicKeyPEM types.String `tfsdk:"public_key_pem"`
	KeyAlgorithm types.String `tfsdk:"key_algorithm"`
	Label        types.String `tfsdk:"label"`
	CreatedAt    types.String `tfsdk:"created_at"`
	Revoked      types.Bool   `tfsdk:"revoked"`
	RevokedAt    types.String `tfsdk:"revoked_at"`
}

func (r *instanceCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_credential"
}

func (r *instanceCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Trufo **instance credential** — a public key enrolled for an instance " +
			"(`POST /gproduct/instance/credential/register`). The matching private key signs the client assertion that " +
			"requests CSR JWTs for certificate enrollment; the private key never leaves your systems. " +
			"An instance may hold at most **two active credentials**: register the replacement, cut over, then revoke the old one. " +
			"Deleting this resource revokes the credential (permanent).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Credential ID (`gpic_…`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gpi_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Instance ID (`gpi_…`) to enroll the key on. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key_pem": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "PEM-encoded public key. Changing this forces recreation (a new registration).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_algorithm": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "JWA algorithm matching the key: `ES256` (EC P-256) or `EdDSA` (Ed25519). Changing this forces recreation.",
				Validators: []validator.String{
					stringvalidator.OneOf(trufo.KeyAlgorithmES256, trufo.KeyAlgorithmEdDSA),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Human-readable label for the credential.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Registration timestamp (RFC 3339).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"revoked": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the credential has been revoked.",
			},
			"revoked_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Revocation timestamp (RFC 3339), if revoked.",
			},
		},
	}
}

func (r *instanceCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	label := plan.Label.ValueString()
	gpicID, err := client.RegisterCredential(ctx, plan.GPIID.ValueString(), plan.PublicKeyPEM.ValueString(), plan.KeyAlgorithm.ValueString(), label)
	if err != nil {
		resp.Diagnostics.AddError("Failed to register instance credential", err.Error())
		return
	}
	plan.ID = types.StringValue(gpicID)

	cred, err := findCredential(ctx, client, plan.GPIID.ValueString(), gpicID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read credential after register", err.Error())
		return
	}
	setCredentialState(&plan, cred)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	cred, err := findCredential(ctx, client, state.GPIID.ValueString(), state.ID.ValueString())
	if err != nil {
		if trufo.Is(err, 404, "") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read instance credential", err.Error())
		return
	}
	if cred == nil {
		// Not in the instance's list anymore: gone.
		resp.State.RemoveResource(ctx)
		return
	}
	setCredentialState(&state, cred)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *instanceCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state instanceCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	if !plan.Label.Equal(state.Label) {
		if err := client.EditCredential(ctx, state.ID.ValueString(), plan.Label.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to edit credential label", err.Error())
			return
		}
	}
	plan.ID = state.ID
	plan.GPIID = state.GPIID

	cred, err := findCredential(ctx, client, state.GPIID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read credential after update", err.Error())
		return
	}
	if cred == nil {
		resp.Diagnostics.AddError("Credential disappeared", "The credential is no longer listed for the instance.")
		return
	}
	setCredentialState(&plan, cred)
	// public_key_pem is not returned by the API; keep the configured value.
	plan.PublicKeyPEM = state.PublicKeyPEM

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	// Deleting the resource revokes the credential. Revocation is permanent.
	if err := client.RevokeCredential(ctx, state.ID.ValueString()); err != nil && !trufo.Is(err, 404, "") {
		resp.Diagnostics.AddError("Failed to revoke instance credential", err.Error())
		return
	}
}

// ImportState accepts "{gpi_id}/{gpic_id}" as the import ID.
func (r *instanceCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected \"{gpi_id}/{gpic_id}\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gpi_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// findCredential lists the instance's credentials and returns the one with
// the given gpic_id, or nil if absent. A 404 from the instance endpoints
// (instance gone) is returned as an error.
func findCredential(ctx context.Context, client *trufo.Client, gpiID, gpicID string) (*trufo.Credential, error) {
	creds, err := client.ListCredentials(ctx, gpiID)
	if err != nil {
		return nil, err
	}
	for i := range creds {
		if creds[i].GPICID == gpicID {
			return &creds[i], nil
		}
	}
	return nil, nil
}

func setCredentialState(m *instanceCredentialResourceModel, c *trufo.Credential) {
	m.KeyAlgorithm = types.StringValue(c.KeyAlgorithm)
	m.Label = types.StringValue(c.Label)
	m.CreatedAt = types.StringValue(c.CreatedAt)
	m.Revoked = types.BoolValue(c.Revoked)
	m.RevokedAt = types.StringValue(c.RevokedAt)
}
