// Copyright (c) Christopher Barnes <christopher@barnes.biz>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	certMgr "certMgr/internal/client"
)

var (
	_ resource.Resource                = &certificateResource{}
	_ resource.ResourceWithConfigure   = &certificateResource{}
	_ resource.ResourceWithImportState = &certificateResource{}
)

func NewCertificateResource() resource.Resource {
	return &certificateResource{}
}

type certificateResourceModel struct {
	ID          types.Int64 `tfsdk:"id"`
	Hostname    types.String `tfsdk:"hostname"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type certificateResource struct {
	client *certMgr.Client
}

func (r *certificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (r *certificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an certificate.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Numeric identifier of the certificate.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the certificate.",
				Computed:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "Name of the hostname that belongs to the certificate.",
				Required:    true,
			},
		},
	}
}

func (r *certificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan certificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificate, err := r.client.CreateCertificate(plan.Hostname.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate",
			"Could not create certificate, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.Int64Value(int64(certificate.ID))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state certificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostname := state.Hostname.ValueString()
	certificate, err := r.client.GetCertificate(hostname)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading certMgr certificate",
			"Could not read certMgr certificate for hostname "+hostname+": "+err.Error(),
		)
		return
	}
	
	state.ID = types.Int64Value(int64(certificate.ID))
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *certificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan certificateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cert certMgr.Certificate
	cert.Hostname = plan.Hostname.String()

	_, err := r.client.UpdateCertificate(plan.Hostname.String(), cert)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating certMgr certificate",
			"Could not update certificate, unexpected error: "+err.Error(),
		)
		return
	}

	certificate, err := r.client.GetCertificate(plan.Hostname.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading certMgr certificate",
			"Could not read certMgr certificate ID "+plan.Hostname.String()+": "+err.Error(),
		)
		return
	}

	plan.Hostname = types.StringValue(certificate.Hostname)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *certificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueInt64()
	if err := r.client.DeleteCertificate(id); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting certMgr certificate",
			fmt.Sprintf("Could not delete certificate with ID %d: %s", id, err.Error()),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *certificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*certMgr.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *certMgr.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *certificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
