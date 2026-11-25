package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	trufo "github.com/mistral/trufo-go"
)

var (
	_ resource.Resource                = (*aiSystemResource)(nil)
	_ resource.ResourceWithImportState = (*aiSystemResource)(nil)
)

// NewAISystemResource creates a trufo_ai_system resource.
func NewAISystemResource() resource.Resource { return &aiSystemResource{} }

type aiSystemResource struct {
	client *trufo.Client
}

func (r *aiSystemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type aiSystemResourceModel struct {
	ID                       types.String  `tfsdk:"id"`
	Name                     types.String  `tfsdk:"name"`
	Version                  types.String  `tfsdk:"version"`
	AIInvolvementModes       types.List    `tfsdk:"ai_involvement_modes"`
	C2PAFirstPartyAttestable types.Bool    `tfsdk:"c2pa_first_party_attestable"`
	C2PAAIDisclosure         types.Dynamic `tfsdk:"c2pa_ai_disclosure"`
	Nickname                 types.String  `tfsdk:"nickname"`
	Status                   types.String  `tfsdk:"status"`
}

func (r *aiSystemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_system"
}

func (r *aiSystemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A registered Trufo **AI system** (`POST /ai-system/add`), referenced by `ai_system_id` when signing " +
			"and watermarking. Registration details are immutable after creation — register each new version as a new AI system. " +
			"Deleting this resource sets the AI system `inactive` (there is no delete endpoint). " +
			"Note: the AI system endpoints are served by the production hosts only, and need a `c2pa-sign-*`/`watermark-*` " +
			"scoped API key or a Bearer token with the `c2pa_sign` permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "AI system ID (`aisys_…`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The AI system's name; with `version`, its C2PA `softwareAgent`. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The AI system's version; a new version is a new registration. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ai_involvement_modes": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Which of `ai_generated` and `ai_modified` the AI system supports (one or both). Changing this forces recreation.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(trufo.AIInvolvementGenerated, trufo.AIInvolvementModified)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"c2pa_first_party_attestable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether the organization operates the AI system and is accountable for its AI labels. " +
					"Requires the Created Assertion Acknowledgement (accepted in the dashboard). Default: `false`. Changing this forces recreation.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"c2pa_ai_disclosure": schema.DynamicAttribute{
				Optional: true,
				MarkdownDescription: "A `c2pa.ai-disclosure` body (C2PA 2.4 § 18.28) describing the underlying model, as a dynamic value. " +
					"Changing this forces recreation.",
			},
			"nickname": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display label only. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "`active` or `inactive`. `inactive` rejects new signs and embeds that use the AI system. Default: `active`.",
				Validators: []validator.String{
					stringvalidator.OneOf(trufo.AISystemStatusActive, trufo.AISystemStatusInactive),
				},
			},
		},
	}
}

func (r *aiSystemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aiSystemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	sys := trufo.AISystem{
		Name:                     plan.Name.ValueString(),
		Version:                  plan.Version.ValueString(),
		C2PAFirstPartyAttestable: plan.C2PAFirstPartyAttestable.ValueBool(),
		Nickname:                 plan.Nickname.ValueString(),
	}
	var modes []string
	resp.Diagnostics.Append(plan.AIInvolvementModes.ElementsAs(ctx, &modes, false)...)
	sys.AIInvolvementModes = modes
	if !plan.C2PAAIDisclosure.IsNull() && !plan.C2PAAIDisclosure.IsUnknown() {
		tv, err := plan.C2PAAIDisclosure.ToTerraformValue(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Invalid c2pa_ai_disclosure", err.Error())
			return
		}
		var disclosure map[string]any
		if err := tv.As(&disclosure); err != nil {
			resp.Diagnostics.AddError("Invalid c2pa_ai_disclosure", err.Error())
			return
		}
		sys.C2PAAIDisclosure = disclosure
	}

	id, err := client.AddAISystem(ctx, sys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to register AI system", err.Error())
		return
	}
	plan.ID = types.StringValue(id)

	if !plan.Status.IsNull() && !plan.Status.IsUnknown() && plan.Status.ValueString() != trufo.AISystemStatusActive {
		if _, err := client.SetAISystemStatus(ctx, id, plan.Status.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to set AI system status", err.Error())
			return
		}
	} else {
		plan.Status = types.StringValue(trufo.AISystemStatusActive)
	}

	stored, err := findAISystem(ctx, client, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read AI system after register", err.Error())
		return
	}
	setAISystemState(&plan, stored)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aiSystemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aiSystemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	stored, err := findAISystem(ctx, client, state.ID.ValueString())
	if err != nil {
		if trufo.Is(err, 404, "") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read AI system", err.Error())
		return
	}
	if stored == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	setAISystemState(&state, stored)
	state.Status = types.StringValue(stored.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aiSystemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state aiSystemResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	if !plan.Status.Equal(state.Status) {
		if _, err := client.SetAISystemStatus(ctx, state.ID.ValueString(), plan.Status.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to set AI system status", err.Error())
			return
		}
	}
	plan.ID = state.ID

	stored, err := findAISystem(ctx, client, state.ID.ValueString())
	if err != nil || stored == nil {
		if err != nil {
			resp.Diagnostics.AddError("Failed to read AI system after update", err.Error())
			return
		}
		resp.Diagnostics.AddError("AI system disappeared", "The AI system is no longer listed.")
		return
	}
	setAISystemState(&plan, stored)
	plan.Status = types.StringValue(stored.Status)
	// Immutable registration fields come from prior state.
	plan.Name = state.Name
	plan.Version = state.Version
	plan.AIInvolvementModes = state.AIInvolvementModes
	plan.C2PAFirstPartyAttestable = state.C2PAFirstPartyAttestable
	plan.C2PAAIDisclosure = state.C2PAAIDisclosure
	plan.Nickname = state.Nickname

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aiSystemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aiSystemResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	// There is no delete endpoint: withdraw the AI system by setting it
	// inactive (idempotent; already-signed content keeps resolving).
	if _, err := client.SetAISystemStatus(ctx, state.ID.ValueString(), trufo.AISystemStatusInactive); err != nil && !trufo.Is(err, 404, "") {
		resp.Diagnostics.AddError("Failed to deactivate AI system", err.Error())
		return
	}
}

func (r *aiSystemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// findAISystem lists AI systems and returns the one with the given id, or nil.
func findAISystem(ctx context.Context, client *trufo.Client, aiSystemID string) (*trufo.AISystem, error) {
	systems, err := client.ListAISystems(ctx)
	if err != nil {
		return nil, err
	}
	for i := range systems {
		if systems[i].AISystemID == aiSystemID {
			return &systems[i], nil
		}
	}
	return nil, nil
}

func setAISystemState(m *aiSystemResourceModel, s *trufo.AISystem) {
	m.Name = types.StringValue(s.Name)
	m.Version = types.StringValue(s.Version)
	modes := make([]attr.Value, 0, len(s.AIInvolvementModes))
	for _, mode := range s.AIInvolvementModes {
		modes = append(modes, types.StringValue(mode))
	}
	m.AIInvolvementModes = types.ListValueMust(types.StringType, modes)
	m.C2PAFirstPartyAttestable = types.BoolValue(s.C2PAFirstPartyAttestable)
	m.Nickname = types.StringValue(s.Nickname)
}
