// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pinpointsmsvoicev2

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkid "github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_pinpointsmsvoicev2_phone_pool", name="Phone Pool")
// @Tags(identifierAttribute="arn")
func newPhonePoolResource(context.Context) (resource.ResourceWithConfigure, error) {
	r := &phonePoolResource{}

	r.SetDefaultCreateTimeout(30 * time.Minute)
	r.SetDefaultUpdateTimeout(30 * time.Minute)
	r.SetDefaultDeleteTimeout(30 * time.Minute)

	return r, nil
}

type phonePoolResource struct {
	framework.ResourceWithConfigure
	framework.WithImportByID
	framework.WithTimeouts
}

func (*phonePoolResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_pinpointsmsvoicev2_phone_pool"
}

func (r *phonePoolResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"deletion_protection_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"opt_out_list_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Default"),
			},
			"origination_identities": schema.SetAttribute{
				CustomType: fwtypes.SetOfARNType,
				Required:   true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"self_managed_opt_outs_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"shared_routes_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"two_way_channel_arn": schema.StringAttribute{
				CustomType: fwtypes.ARNType,
				Optional:   true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("two_way_channel_enabled"),
					),
				},
			},
			"two_way_channel_role": schema.StringAttribute{
				CustomType: fwtypes.ARNType,
				Optional:   true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("two_way_channel_enabled"),
					),
				},
			},
			"two_way_channel_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"message_type": schema.StringAttribute{
				CustomType: fwtypes.StringEnumType[awstypes.MessageType](),
				Required:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrID:      framework.IDAttribute(),
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
		Blocks: map[string]schema.Block{
			names.AttrTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *phonePoolResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data phonePoolResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	createOriginationIdentity := data.OriginationIdentities.Elements()[0].String()
	var createCountryCode string
	var err error
	if isOriginationIdentitityASenderID(createOriginationIdentity) {
		createCountryCode, _ = getIsoCountryCodeForSenderIDARN(createOriginationIdentity)
	} else {
		createCountryCode, err = getIsoCountryCodeForPhoneARN(ctx, conn, createOriginationIdentity)
		if err != nil {
			response.Diagnostics.AddError("Error getting ISO country code for phone ARN", err.Error())
			return
		}
	}

	input := &pinpointsmsvoicev2.CreatePoolInput{
		ClientToken:               aws.String(sdkid.UniqueId()),
		DeletionProtectionEnabled: fwflex.BoolFromFramework(ctx, data.DeletionProtectionEnabled),
		// The CreatePool API request requires at least one country code to be specified during creation.
		IsoCountryCode:      aws.String(createCountryCode),
		OriginationIdentity: aws.String(createOriginationIdentity),
		MessageType:         awstypes.MessageType(data.MessageType.ValueString()),
		Tags:                getTagsIn(ctx),
	}

	output, err := conn.CreatePool(ctx, input)

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("creating End User Messaging SMS Phone Pool (%s)", data.ID.ValueString()), err.Error())
		return
	}

	data.ID = types.StringValue(aws.ToString(output.PoolId))
	data.PhonePoolARN = fwflex.StringToFramework(ctx, output.PoolArn)

	out, err := waitPhonePoolActive(ctx, conn, data.ID.ValueString(), r.CreateTimeout(ctx, data.Timeouts))

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("waiting for End User Messaging SMS Phone Pool (%s) create", data.ID.ValueString()), err.Error())

		return
	}

	for _, originationIdentity := range data.OriginationIdentities.Elements() {
		var countryCode string
		// The initial origination identity is already added to the pool
		if originationIdentity.String() != createOriginationIdentity {
			if isOriginationIdentitityASenderID(createOriginationIdentity) {
				countryCode, _ = getIsoCountryCodeForSenderIDARN(originationIdentity.String())
			} else {
				countryCode, err = getIsoCountryCodeForPhoneARN(ctx, conn, originationIdentity.String())
				if err != nil {
					response.Diagnostics.AddError("Error getting ISO country code for phone ARN", err.Error())
					return
				}
			}

			input := &pinpointsmsvoicev2.AssociateOriginationIdentityInput{
				ClientToken:         aws.String(sdkid.UniqueId()),
				IsoCountryCode:      aws.String(countryCode),
				OriginationIdentity: aws.String(originationIdentity.String()),
				PoolId:              data.ID.ValueStringPointer(),
			}

			_, err := conn.AssociateOriginationIdentity(ctx, input)
			if err != nil {
				response.Diagnostics.AddError(fmt.Sprintf("associating origination identity (%s) to End User Messaging SMS Phone Pool (%s)", originationIdentity.String(), data.ID.ValueString()), err.Error())
				return
			}
		}
	}

	if !data.DeletionProtectionEnabled.IsNull() || !data.OptOutListName.IsNull() || !data.ID.IsNull() || !data.SelfManagedOptOutsEnabled.IsNull() || !data.SharedRoutesEnabled.IsNull() || !data.TwoWayChannelARN.IsNull() || !data.TwoWayChannelRole.IsNull() || !data.TwoWayEnabled.IsNull() {
		input := &pinpointsmsvoicev2.UpdatePoolInput{
			DeletionProtectionEnabled: fwflex.BoolFromFramework(ctx, data.DeletionProtectionEnabled),
			OptOutListName:            fwflex.StringFromFramework(ctx, data.OptOutListName),
			PoolId:                    fwflex.StringFromFramework(ctx, data.ID),
			SelfManagedOptOutsEnabled: fwflex.BoolFromFramework(ctx, data.SelfManagedOptOutsEnabled),
			SharedRoutesEnabled:       fwflex.BoolFromFramework(ctx, data.SharedRoutesEnabled),
			TwoWayChannelArn:          fwflex.StringFromFramework(ctx, data.TwoWayChannelARN),
			TwoWayChannelRole:         fwflex.StringFromFramework(ctx, data.TwoWayChannelRole),
			TwoWayEnabled:             fwflex.BoolFromFramework(ctx, data.TwoWayEnabled),
		}

		_, err := conn.UpdatePool(ctx, input)

		if err != nil {
			response.Diagnostics.AddError(fmt.Sprintf("updating End User Messaging Phone Pool (%s)", data.ID.ValueString()), err.Error())

			return
		}
	}

	response.Diagnostics.Append(fwflex.Flatten(ctx, out, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (r *phonePoolResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data phonePoolResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	out, err := findPhonePoolById(ctx, conn, data.ID.ValueString())

	if tfresource.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)

		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("reading End User Messaging SMS Phone Pool (%s)", data.ID.ValueString()), err.Error())
		return
	}

	response.Diagnostics.Append(fwflex.Flatten(ctx, out, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *phonePoolResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var old, new phonePoolResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &old)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	if !new.DeletionProtectionEnabled.Equal(old.DeletionProtectionEnabled) || !new.OptOutListName.Equal(old.OptOutListName) || !new.ID.Equal(old.ID) || !new.SelfManagedOptOutsEnabled.Equal(old.SelfManagedOptOutsEnabled) || !new.SharedRoutesEnabled.Equal(old.SharedRoutesEnabled) || !new.TwoWayChannelARN.Equal(old.TwoWayChannelARN) || !new.TwoWayChannelRole.Equal(old.TwoWayChannelRole) || !new.TwoWayEnabled.Equal(old.TwoWayEnabled) {
		input := &pinpointsmsvoicev2.UpdatePoolInput{
			DeletionProtectionEnabled: fwflex.BoolFromFramework(ctx, new.DeletionProtectionEnabled),
			OptOutListName:            fwflex.StringFromFramework(ctx, new.OptOutListName),
			PoolId:                    fwflex.StringFromFramework(ctx, new.ID),
			SelfManagedOptOutsEnabled: fwflex.BoolFromFramework(ctx, new.SelfManagedOptOutsEnabled),
			SharedRoutesEnabled:       fwflex.BoolFromFramework(ctx, new.SharedRoutesEnabled),
			TwoWayChannelArn:          fwflex.StringFromFramework(ctx, new.TwoWayChannelARN),
			TwoWayChannelRole:         fwflex.StringFromFramework(ctx, new.TwoWayChannelRole),
			TwoWayEnabled:             fwflex.BoolFromFramework(ctx, new.TwoWayEnabled),
		}

		_, err := conn.UpdatePool(ctx, input)

		if err != nil {
			response.Diagnostics.AddError(fmt.Sprintf("updating End User Messaging Phone Pool (%s)", new.ID.ValueString()), err.Error())

			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *phonePoolResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data phonePoolResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().PinpointSMSVoiceV2Client(ctx)

	_, err := conn.DeletePool(ctx, &pinpointsmsvoicev2.DeletePoolInput{
		PoolId: data.ID.ValueStringPointer(),
	})

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("deleting End User Messaging SMS Phone Pool (%s)", data.ID.ValueString()), err.Error())
		return
	}
}

func (r *phonePoolResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	r.SetTagsAll(ctx, request, response)
}

type phonePoolResourceModel struct {
	ID                        types.String                             `tfsdk:"id"`
	PhonePoolARN              types.String                             `tfsdk:"arn"`
	DeletionProtectionEnabled types.Bool                               `tfsdk:"deletion_protection_enabled"`
	OptOutListName            types.String                             `tfsdk:"opt_out_list_name"`
	SelfManagedOptOutsEnabled types.Bool                               `tfsdk:"self_managed_opt_outs_enabled"`
	SharedRoutesEnabled       types.Bool                               `tfsdk:"shared_routes_enabled"`
	TwoWayChannelARN          fwtypes.ARN                              `tfsdk:"two_way_channel_arn"`
	TwoWayChannelRole         fwtypes.ARN                              `tfsdk:"two_way_channel_role"`
	TwoWayEnabled             types.Bool                               `tfsdk:"two_way_channel_enabled"`
	MessageType               fwtypes.StringEnum[awstypes.MessageType] `tfsdk:"message_type"`
	Timeouts                  timeouts.Value                           `tfsdk:"timeouts"`
	OriginationIdentities     fwtypes.SetValueOf[fwtypes.ARN]          `tfsdk:"origination_identities"`
	Tags                      tftags.Map                               `tfsdk:"tags"`
	TagsAll                   tftags.Map                               `tfsdk:"tags_all"`
}

func findPhonePoolById(ctx context.Context, conn *pinpointsmsvoicev2.Client, id string) (*awstypes.PoolInformation, error) {
	input := &pinpointsmsvoicev2.DescribePoolsInput{
		PoolIds: []string{id},
	}

	return findPhonePool(ctx, conn, input)
}

func findPhonePool(ctx context.Context, conn *pinpointsmsvoicev2.Client, input *pinpointsmsvoicev2.DescribePoolsInput) (*awstypes.PoolInformation, error) {
	output, err := findPhonePools(ctx, conn, input)

	if err != nil {
		return nil, err
	}

	return tfresource.AssertSingleValueResult(output)
}

func findPhonePools(ctx context.Context, conn *pinpointsmsvoicev2.Client, input *pinpointsmsvoicev2.DescribePoolsInput) ([]awstypes.PoolInformation, error) {
	var output []awstypes.PoolInformation

	pages := pinpointsmsvoicev2.NewDescribePoolsPaginator(conn, input)
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

		output = append(output, page.Pools...)
	}

	return output, nil
}

func statusPhonePool(ctx context.Context, conn *pinpointsmsvoicev2.Client, id string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := findPhonePoolById(ctx, conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, string(output.Status), nil
	}
}

func waitPhonePoolActive(ctx context.Context, conn *pinpointsmsvoicev2.Client, id string, timeout time.Duration) (*awstypes.PoolInformation, error) {
	stateConf := &retry.StateChangeConf{
		Pending: enum.Slice(awstypes.PoolStatusCreating),
		Target:  enum.Slice(awstypes.NumberStatusActive),
		Refresh: statusPhonePool(ctx, conn, id),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.PoolInformation); ok {
		return output, err
	}

	return nil, err
}

func waitPhonePoolDeleted(ctx context.Context, conn *pinpointsmsvoicev2.Client, id string, timeout time.Duration) (*awstypes.PoolInformation, error) {
	stateConf := &retry.StateChangeConf{
		Pending: enum.Slice(awstypes.PoolStatusDeleting),
		Target:  []string{},
		Refresh: statusPhonePool(ctx, conn, id),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.PoolInformation); ok {
		return output, err
	}

	return nil, err
}

func getIsoCountryCodeForPhoneARN(ctx context.Context, conn *pinpointsmsvoicev2.Client, id string) (string, error) {
	out, err := findPhoneNumberByID(ctx, conn, id)

	if err != nil {
		return "", err
	}

	return *out.IsoCountryCode, nil
}

func isOriginationIdentitityASenderID(originationIdentity string) bool {
	return strings.Contains(originationIdentity, ":sender-id/")
}

func getIsoCountryCodeForSenderIDARN(id string) (string, error) {
	if !isOriginationIdentitityASenderID(id) {
		return "", errors.New("the origination identity is not a sender ID")
	}
	//The country code of a sender ID ARN is the last two characters of the ARN.
	return id[len(id)-2:], nil
}
