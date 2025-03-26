// Copyright (c) HashiCorp, Inc.

package provider

import (
	certMgr "certMgr/internal/client"
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ provider.Provider = &certMgrProvider{}
)

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &certMgrProvider{
			version: version,
		}
	}
}

type certMgrProviderModel struct {
	Host types.String `tfsdk:"host"`
	Port types.String `tfsdk:"port"`
}

type certMgrProvider struct {
	version string
}

func (p *certMgrProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "certmgr"
	resp.Version = p.version
}

func (p *certMgrProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with certMgr.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "URI for certMgr API. May also be provided via CERTMGR_HOST environment variable.",
				Optional:    true,
			},
			"port": schema.StringAttribute{
				Description: "Port for certMgr API. May also be provided via CERTMGR_PORT environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *certMgrProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring certMgr client")

	var config certMgrProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown certMgr API Host",
			"The provider cannot create the certMgr API client as there is an unknown configuration value for the certMgr host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the certMgr_HOST environment variable.",
		)
	}

	if config.Port.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Unknown certMgr host Port",
			"The provider cannot create the certMgr API client as there is an unknown configuration value for the certMgr port. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CERTMGR_PORT environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("CERTMGR_HOST")
	port := os.Getenv("CERTMGR_PORT")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Port.IsNull() {
		port = config.Port.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing certMgr API Host",
			"The provider cannot create the certMgr API client as there is a missing or empty value for the certMgr host. "+
				"Set the host value in the configuration or use the CERTMGR_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if port == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Missing certMgr Port",
			"The provider cannot create the certMgr API client as there is a missing or empty value for the certMgr port. "+
				"Set the port value in the configuration or use the CERTMGR_PORT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "certMgr_host", host)
	ctx = tflog.SetField(ctx, "certMgr_port", port)

	tflog.Debug(ctx, "Creating certMgr client")

	client, err := certMgr.NewClient(&host, &port)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create certMgr API Client",
			"An unexpected error occurred when creating the certMgr API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"certMgr Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured certMgr client", map[string]any{"success": true})
}

func (p *certMgrProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCertificateResource,
	}
}

func (p *certMgrProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
