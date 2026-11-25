package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	trufo "github.com/NitriKx/terraform-provider-trufo/trufo-go"
)

var _ datasource.DataSource = (*certificatesDataSource)(nil)

// NewCertificatesDataSource creates a trufo_certificates data source.
func NewCertificatesDataSource() datasource.DataSource { return &certificatesDataSource{} }

type certificatesDataSource struct {
	client *trufo.Client
}

func (d *certificatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type certificateModel struct {
	SerialNumber     types.String `tfsdk:"serial_number"`
	LeafType         types.String `tfsdk:"leaf_type"`
	IssueTime        types.String `tfsdk:"issue_time"`
	ExpiryTime       types.String `tfsdk:"expiry_time"`
	RevocationStatus types.String `tfsdk:"revocation_status"`
	RevocationReason types.String `tfsdk:"revocation_reason"`
	RevocationTime   types.String `tfsdk:"revocation_time"`
	Revocable        types.Bool   `tfsdk:"revocable"`
	TrufoManaged     types.Bool   `tfsdk:"trufo_managed"`
	GPIID            types.String `tfsdk:"gpi_id"`
	GPIName          types.String `tfsdk:"gpi_name"`
	GPID             types.String `tfsdk:"gp_id"`
	GPName           types.String `tfsdk:"gp_name"`
}

type certificatesDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Certs types.List   `tfsdk:"certs"`
}

func (d *certificatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificates"
}

func certificateSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"serial_number":     schema.StringAttribute{Computed: true},
		"leaf_type":         schema.StringAttribute{Computed: true},
		"issue_time":        schema.StringAttribute{Computed: true},
		"expiry_time":       schema.StringAttribute{Computed: true},
		"revocation_status": schema.StringAttribute{Computed: true},
		"revocation_reason": schema.StringAttribute{Computed: true},
		"revocation_time":   schema.StringAttribute{Computed: true},
		"revocable":         schema.BoolAttribute{Computed: true},
		"trufo_managed":     schema.BoolAttribute{Computed: true},
		"gpi_id":            schema.StringAttribute{Computed: true},
		"gpi_name":          schema.StringAttribute{Computed: true},
		"gp_id":             schema.StringAttribute{Computed: true},
		"gp_name":           schema.StringAttribute{Computed: true},
	}
}

func (d *certificatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the organization's certificates (`POST /cert/list`). Requires a Bearer access token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable identifier for this list (number of certificates).",
			},
			"certs": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The certificates.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: certificateSchema(),
				},
			},
		},
	}
}

func (d *certificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg certificatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := d.client

	certs, err := client.ListCertificates(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list certificates", err.Error())
		return
	}

	objs := make([]attr.Value, 0, len(certs))
	for _, c := range certs {
		obj, diags := types.ObjectValue(certificateAttrTypes(), map[string]attr.Value{
			"serial_number":     types.StringValue(c.SerialNumber),
			"leaf_type":         types.StringValue(c.LeafType),
			"issue_time":        types.StringValue(c.IssueTime),
			"expiry_time":       types.StringValue(c.ExpiryTime),
			"revocation_status": types.StringValue(c.RevocationStatus),
			"revocation_reason": types.StringValue(c.RevocationReason),
			"revocation_time":   types.StringValue(c.RevocationTime),
			"revocable":         types.BoolValue(c.Revocable),
			"trufo_managed":     types.BoolValue(c.TrufoManaged),
			"gpi_id":            types.StringValue(c.GPIID),
			"gpi_name":          types.StringValue(c.GPIName),
			"gp_id":             types.StringValue(c.GPID),
			"gp_name":           types.StringValue(c.GPName),
		})
		resp.Diagnostics.Append(diags...)
		objs = append(objs, obj)
	}

	// Stable ID: a hash of the serial numbers, so it does not change merely
	// because the certificate count changed.
	h := sha256.New()
	for _, c := range certs {
		h.Write([]byte(c.SerialNumber))
		h.Write([]byte{0})
	}
	cfg.ID = types.StringValue(hex.EncodeToString(h.Sum(nil))[:16])
	cfg.Certs = types.ListValueMust(certificateObjectType(), objs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

func certificateAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"serial_number":     types.StringType,
		"leaf_type":         types.StringType,
		"issue_time":        types.StringType,
		"expiry_time":       types.StringType,
		"revocation_status": types.StringType,
		"revocation_reason": types.StringType,
		"revocation_time":   types.StringType,
		"revocable":         types.BoolType,
		"trufo_managed":     types.BoolType,
		"gpi_id":            types.StringType,
		"gpi_name":          types.StringType,
		"gp_id":             types.StringType,
		"gp_name":           types.StringType,
	}
}

func certificateObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: certificateAttrTypes()}
}
