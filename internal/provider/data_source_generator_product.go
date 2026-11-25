package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	trufo "github.com/NitriKx/terraform-trufo-ai-provider/trufo-go"
)

var _ datasource.DataSource = (*generatorProductDataSource)(nil)

// NewGeneratorProductDataSource creates a trufo_generator_product data source.
func NewGeneratorProductDataSource() datasource.DataSource { return &generatorProductDataSource{} }

type generatorProductDataSource struct {
	client *trufo.Client
}

func (d *generatorProductDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*trufo.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected *trufo.Client.")
		return
	}
	d.client = c
}

type generatorProductDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	GPID               types.String `tfsdk:"gp_id"`
	Name               types.String `tfsdk:"name"`
	OID                types.String `tfsdk:"oid"`
	Implementation     types.String `tfsdk:"implementation"`
	PVStatus           types.String `tfsdk:"pv_status"`
	PVActive           types.Bool   `tfsdk:"pv_active"`
	ValidatedExpiresAt types.String `tfsdk:"validated_expires_at"`
	CancelAt           types.String `tfsdk:"cancel_at"`
	CreatedBy          types.String `tfsdk:"created_by"`
	CreatedAt          types.String `tfsdk:"created_at"`
}

func (d *generatorProductDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generator_product"
}

func (d *generatorProductDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Trufo **generator product** (`POST /gproduct/list` / `/gproduct/info`). " +
			"Products are created in the Trufo dashboard (as part of a Product Validation application), so this is read-only. " +
			"Look up by `gp_id`, or by `name` (then `oid` is passed to the list call, which the server scopes to your credential's organization).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Generator product ID (`gp_…`).",
			},
			"gp_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Generator product ID (`gp_…`) to look up directly.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Product name to match when listing. Exactly one of `gp_id` / `name` must be set.",
			},
			"oid": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Organization ID passed to `/gproduct/list` (required by its request schema; the server uses your credential's organization).",
			},
			"implementation": schema.StringAttribute{
				Computed: true,
			},
			"pv_status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Product Validation status.",
			},
			"pv_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the Product Validation is approved and unexpired — required before instances can be created.",
			},
			"validated_expires_at": schema.StringAttribute{
				Computed: true,
			},
			"cancel_at": schema.StringAttribute{
				Computed: true,
			},
			"created_by": schema.StringAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *generatorProductDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg generatorProductDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := d.client

	// The list endpoint carries the PV status fields; the info endpoint
	// carries oid/created_by/created_at. Merge both.
	oid := cfg.OID.ValueString()
	gps, err := client.ListGProducts(ctx, oid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list generator products", err.Error())
		return
	}
	var listed *trufo.GProduct
	if !cfg.GPID.IsNull() && cfg.GPID.ValueString() != "" {
		for i := range gps {
			if gps[i].GPID == cfg.GPID.ValueString() {
				listed = &gps[i]
				break
			}
		}
	} else if !cfg.Name.IsNull() && cfg.Name.ValueString() != "" {
		for i := range gps {
			if gps[i].Name == cfg.Name.ValueString() {
				listed = &gps[i]
				break
			}
		}
	} else {
		resp.Diagnostics.AddError("Missing lookup key", "Set exactly one of gp_id or name.")
		return
	}
	if listed == nil {
		resp.Diagnostics.AddError("Generator product not found",
			fmt.Sprintf("No generator product matching gp_id=%q name=%q in the list.", cfg.GPID.ValueString(), cfg.Name.ValueString()))
		return
	}
	gp := *listed
	if info, err := client.GetGProduct(ctx, gp.GPID); err == nil {
		// info wins for the fields it actually returns
		gp.OID = info.OID
		gp.CreatedBy = info.CreatedBy
		gp.CreatedAt = info.CreatedAt
		if info.Name != "" {
			gp.Name = info.Name
		}
	}

	cfg.ID = types.StringValue(gp.GPID)
	cfg.GPID = types.StringValue(gp.GPID)
	cfg.Name = types.StringValue(gp.Name)
	cfg.OID = types.StringValue(gp.OID)
	cfg.Implementation = types.StringValue(gp.Implementation)
	cfg.PVStatus = types.StringValue(gp.PVStatus)
	cfg.PVActive = types.BoolValue(gp.PVActive)
	cfg.ValidatedExpiresAt = types.StringValue(gp.ValidatedExpiresAt)
	cfg.CancelAt = types.StringValue(gp.CancelAt)
	cfg.CreatedBy = types.StringValue(gp.CreatedBy)
	cfg.CreatedAt = types.StringValue(gp.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
