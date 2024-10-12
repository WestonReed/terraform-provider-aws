// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package pinpointsmsvoicev2_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfpinpointsmsvoicev2 "github.com/hashicorp/terraform-provider-aws/internal/service/pinpointsmsvoicev2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccPinpointSMSVoiceV2PhonePool_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var configurationSet awstypes.PoolInformation
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_pinpointsmsvoicev2_configuration_set.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			testAccPreCheckPhonePool(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.PinpointSMSVoiceV2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPhonePoolDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccPhonePoolConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPhonePoolExists(ctx, resourceName, &configurationSet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrARN), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrName), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrTags), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrTagsAll), knownvalue.MapExact(map[string]knownvalue.Check{})),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPinpointSMSVoiceV2PhonePool_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var configurationSet awstypes.PoolInformation
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_pinpointsmsvoicev2_configuration_set.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			testAccPreCheckPhonePool(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.PinpointSMSVoiceV2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPhonePoolDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccPhonePoolConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPhonePoolExists(ctx, resourceName, &configurationSet),
					acctest.CheckFrameworkResourceDisappears(ctx, acctest.Provider, tfpinpointsmsvoicev2.ResourcePhonePool, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccPinpointSMSVoiceV2PhonePool_tags(t *testing.T) {
	ctx := acctest.Context(t)
	var configurationSet awstypes.PoolInformation
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_pinpointsmsvoicev2_configuration_set.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			testAccPreCheckPhonePool(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.PinpointSMSVoiceV2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckPhonePoolDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccPhonePoolConfig_tags1(rName, acctest.CtKey1, acctest.CtValue1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPhonePoolExists(ctx, resourceName, &configurationSet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrTags), knownvalue.MapExact(map[string]knownvalue.Check{
						acctest.CtKey1: knownvalue.StringExact(acctest.CtValue1),
					})),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPhonePoolConfig_tags2(rName, acctest.CtKey1, acctest.CtValue1Updated, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPhonePoolExists(ctx, resourceName, &configurationSet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrTags), knownvalue.MapExact(map[string]knownvalue.Check{
						acctest.CtKey1: knownvalue.StringExact(acctest.CtValue1Updated),
						acctest.CtKey2: knownvalue.StringExact(acctest.CtValue2),
					})),
				},
			},
			{
				Config: testAccPhonePoolConfig_tags1(rName, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPhonePoolExists(ctx, resourceName, &configurationSet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrTags), knownvalue.MapExact(map[string]knownvalue.Check{
						acctest.CtKey2: knownvalue.StringExact(acctest.CtValue2),
					})),
				},
			},
		},
	})
}

func testAccCheckPhonePoolDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).PinpointSMSVoiceV2Client(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_pinpointsmsvoicev2_configuration_set" {
				continue
			}

			_, err := tfpinpointsmsvoicev2.FindPhonePoolByID(ctx, conn, rs.Primary.ID)

			if tfresource.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("End User Messaging SMS Configuration Set %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckPhonePoolExists(ctx context.Context, n string, v *awstypes.PoolInformation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).PinpointSMSVoiceV2Client(ctx)

		output, err := tfpinpointsmsvoicev2.FindPoolByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccPreCheckPhonePool(ctx context.Context, t *testing.T) {
	conn := acctest.Provider.Meta().(*conns.AWSClient).PinpointSMSVoiceV2Client(ctx)

	input := &pinpointsmsvoicev2.DescribePhonePoolsInput{}

	_, err := conn.DescribePhonePools(ctx, input)

	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}

	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

func testAccPhonePoolConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_pinpointsmsvoicev2_configuration_set" "test" {
  name = %[1]q
}
`, rName)
}

func testAccPhonePoolConfig_tags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_pinpointsmsvoicev2_configuration_set" "test" {
  name = %[1]q

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccPhonePoolConfig_tags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_pinpointsmsvoicev2_configuration_set" "test" {
  name = %[1]q

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}
