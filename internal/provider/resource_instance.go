package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	trufo "github.com/NitriKx/terraform-provider-trufo-ai/trufo-go"
)

var _ resource.Resource = (*instanceResource)(nil)
var _ resource.ResourceWithImportState = (*instanceResource)(nil)

// NewInstanceResource creates a trufo-ai_instance resource.
func NewInstanceResource() resource.Resource { return &instanceResource{} }

type instanceResource struct {
	client *trufo.Client
}

func (r *instanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type instanceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	GPID      types.String `tfsdk:"gp_id"`
	Name      types.String `tfsdk:"name"`
	OID       types.String `tfsdk:"oid"`
	RecordID  types.String `tfsdk:"record_id"`
	CreatedBy types.String `tfsdk:"created_by"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Trufo generator product **instance** — one signing identity within a generator product " +
			"(`POST /gproduct/instance/*`). Instances hold their own credentials and are issued their own certificates. " +
			"Requires an active Product Validation on the product.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Instance ID (`gpi_…`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gp_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Generator product ID (`gp_…`) the instance belongs to. Changing this forces recreation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable instance name.",
			},
			"oid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Owning organization ID (`org_…`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"record_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Product record ID, embedded in certificates issued to this instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creator of the instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp (RFC 3339).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	gpiID, err := client.CreateInstance(ctx, plan.GPID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}
	plan.ID = types.StringValue(gpiID)

	inst, err := client.GetInstance(ctx, gpiID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance after create", err.Error())
		return
	}
	plan.GPID = types.StringValue(inst.GPID)
	plan.Name = types.StringValue(inst.Name)
	plan.OID = types.StringValue(inst.OID)
	plan.RecordID = types.StringValue(inst.RecordID)
	plan.CreatedBy = types.StringValue(inst.CreatedBy)
	plan.CreatedAt = types.StringValue(inst.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	inst, err := client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		if trufo.Is(err, 404, "") {
			// Instance is gone; drop it from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}
	state.GPID = types.StringValue(inst.GPID)
	state.Name = types.StringValue(inst.Name)
	state.OID = types.StringValue(inst.OID)
	state.RecordID = types.StringValue(inst.RecordID)
	state.CreatedBy = types.StringValue(inst.CreatedBy)
	state.CreatedAt = types.StringValue(inst.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state instanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	if !plan.Name.Equal(state.Name) {
		if err := client.RenameInstance(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename instance", err.Error())
			return
		}
	}
	plan.ID = state.ID
	plan.GPID = state.GPID

	inst, err := client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance after update", err.Error())
		return
	}
	plan.Name = types.StringValue(inst.Name)
	plan.OID = types.StringValue(inst.OID)
	plan.RecordID = types.StringValue(inst.RecordID)
	plan.CreatedBy = types.StringValue(inst.CreatedBy)
	plan.CreatedAt = types.StringValue(inst.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client

	if err := client.DeleteInstance(ctx, state.ID.ValueString()); err != nil && !trufo.Is(err, 404, "") {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
