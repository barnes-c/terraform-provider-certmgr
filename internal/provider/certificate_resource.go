// Copyright (c) Christopher Barnes <christopher.barnes@cern.ch>
// SPDX-License-Identifier: GPL-3.0-or-later

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
	ID          types.Int64  `tfsdk:"id"`
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
		Description: "Manages a certificate.",
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
				Description: "Hostname that the certificate belongs to.",
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

	certificate, err := r.client.CreateCertificate(plan.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate",
			"Could not create certificate: "+err.Error(),
		)
		return
	}

	plan.ID = types.Int64Value(int64(certificate.ID))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
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
			"Error Reading certificate",
			fmt.Sprintf("Could not read certificate for hostname %s: %s", hostname, err),
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

	certificate, err := r.client.GetCertificate(plan.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching certificate",
			"Could not fetch certificate for update: "+err.Error(),
		)
		return
	}

	err = r.client.UpdateCertificate(*certificate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating certificate",
			"Could not update certificate: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *certificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certificateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hostname := state.Hostname.ValueString()
	if err := r.client.DeleteCertificate(hostname); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting certificate",
			fmt.Sprintf("Could not delete certificate for hostname %s: %s", hostname, err),
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
			"Unexpected client type",
			fmt.Sprintf("Expected *certMgr.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *certificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
