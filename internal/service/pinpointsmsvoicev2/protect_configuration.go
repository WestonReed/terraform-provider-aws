// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pinpointsmsvoicev2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkid "github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_pinpointsmsvoicev2_protect_configuration", name="Protect Configuration")
// @Tags(identifierAttribute="arn")
func newProtectConfigurationResource(context.Context) (resource.ResourceWithConfigure, error) {
	r := &protectConfigurationResource{}

	return r, nil
}

type protectConfigurationResource struct {
	framework.ResourceWithConfigure
	framework.WithImportByID
}

func (*protectConfigurationResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_pinpointsmsvoicev2_protect_configuration"
}

func (r *protectConfigurationResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"account_default": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"deletion_protection_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			names.AttrID:      framework.IDAttribute(),
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
	}
}

func (r *protectConfigurationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data protectConfigurationResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	input := &pinpointsmsvoicev2.CreateProtectConfigurationInput{
		ClientToken:               aws.String(sdkid.UniqueId()),
		DeletionProtectionEnabled: aws.Bool(name),
		Tags:                      getTagsIn(ctx),
	}

	output, err := conn.CreateProtectConfiguration(ctx, input)

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("creating End User Messaging SMS Protect Configuration (%s)", name), err.Error())

		return
	}

	// Set values for unknowns.
	data.ProtectConfigurationARN = fwflex.StringToFramework(ctx, output.ProtectConfigurationArn)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (r *protectConfigurationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data protectConfigurationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := data.InitFromID(); err != nil {
		response.Diagnostics.AddError("parsing resource ID", err.Error())

		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	out, err := findProtectConfigurationByID(ctx, conn, data.ID.ValueString())

	if tfresource.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)

		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("reading End User Messaging SMS Protect Configuration (%s)", data.ID.ValueString()), err.Error())

		return
	}

	// Set attributes for import.
	response.Diagnostics.Append(fwflex.Flatten(ctx, out, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *protectConfigurationResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var old, new protectConfigurationResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &old)...)
	if response.Diagnostics.HasError() {
		return
	}

	//conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *protectConfigurationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data protectConfigurationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	_, err := conn.DeleteProtectConfiguration(ctx, &pinpointsmsvoicev2.DeleteProtectConfigurationInput{
		ProtectConfigurationId: data.ID.ValueStringPointer(),
	})

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("deleting End User Messaging SMS Protect Configuration (%s)", data.ID.ValueString()), err.Error())

		return
	}
}

func (r *protectConfigurationResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	r.SetTagsAll(ctx, request, response)
}

type protectConfigurationResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	ProtectConfigurationARN   types.String `tfsdk:"arn"`
	DeletionProtectionEnabled types.Bool   `tfsdk:"deletion_protection_enabled"`
	Tags                      tftags.Map   `tfsdk:"tags"`
	TagsAll                   tftags.Map   `tfsdk:"tags_all"`
}

func findProtectConfigurationByID(ctx context.Context, conn *pinpointsmsvoicev2.Client, id string) (*awstypes.ProtectConfigurationInformation, error) {
	input := &pinpointsmsvoicev2.DescribeProtectConfigurationsInput{
		ProtectConfigurationIds: []string{id},
	}

	return findProtectConfiguration(ctx, conn, input)
}

func findProtectConfiguration(ctx context.Context, conn *pinpointsmsvoicev2.Client, input *pinpointsmsvoicev2.DescribeProtectConfigurationsInput) (*awstypes.ProtectConfigurationInformation, error) {
	output, err := findProtectConfigurations(ctx, conn, input)

	if err != nil {
		return nil, err
	}

	return tfresource.AssertSingleValueResult(output)
}

func findProtectConfigurations(ctx context.Context, conn *pinpointsmsvoicev2.Client, input *pinpointsmsvoicev2.DescribeProtectConfigurationsInput) ([]awstypes.ProtectConfigurationInformation, error) {
	var output []awstypes.ProtectConfigurationInformation

	pages := pinpointsmsvoicev2.NewDescribeProtectConfigurationsPaginator(conn, input)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)

		if errs.IsA[*awstypes.ResourceNotFoundException](err) {
			return nil, &retry.NotFoundError{
				LastError:   err,
				LastRequest: input,
			}
		}

		if err != nil {
			return nil, err
		}

		output = append(output, page.ProtectConfigurations...)
	}

	return output, nil
}
