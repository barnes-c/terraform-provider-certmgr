// Copyright (c) Christopher Barnes <christopher@barnes.biz>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	certMgr "certMgr/internal/client"
	"context"
	"fmt"
	"os"
	"strconv"

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
	Port types.Number `tfsdk:"port"`
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
			"port": schema.NumberAttribute{
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
	portStr := os.Getenv("CERTMGR_PORT")
	port := 0
	
	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}
	
	if !config.Port.IsNull() {
		bf := config.Port.ValueBigFloat()
		portInt64, _ := bf.Int64()
		port = int(portInt64)
	} else if portStr != "" {
		parsed, err := strconv.Atoi(portStr)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("port"),
				"Invalid ROGER_PORT Environment Variable",
				fmt.Sprintf("ROGER_PORT must be an integer, but got: %q", portStr),
			)
			return
		}
		port = parsed
	}
	
	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing certMgr Host",
			"Set the host value in the configuration or via the CERTMGR_HOST environment variable.",
		)
	}
	
	if port == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Missing certMgr Port",
			"Set the port value in the configuration or via the CERTMGR_PORT environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "certMgr_host", host)
	ctx = tflog.SetField(ctx, "certMgr_port", port)

	tflog.Debug(ctx, "Creating certMgr client")

	client, err := certMgr.NewClient(host, port)
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
