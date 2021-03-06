package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMParameter_basic(t *testing.T) {
	var param ssm.Parameter
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterBasicConfig(name, "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.foo", &param),
					resource.TestCheckResourceAttr("aws_ssm_parameter.foo", "value", "bar"),
					resource.TestCheckResourceAttr("aws_ssm_parameter.foo", "type", "String"),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_disappears(t *testing.T) {
	var param ssm.Parameter
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterBasicConfig(name, "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.foo", &param),
					testAccCheckAWSSSMParameterDisappears(&param),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSSSMParameter_update(t *testing.T) {
	var param ssm.Parameter
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterBasicConfig(name, "bar"),
			},
			{
				Config: testAccAWSSSMParameterBasicConfigOverwrite(name, "baz1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.foo", &param),
					resource.TestCheckResourceAttr("aws_ssm_parameter.foo", "value", "baz1"),
					resource.TestCheckResourceAttr("aws_ssm_parameter.foo", "type", "String"),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_changeNameForcesNew(t *testing.T) {
	var beforeParam, afterParam ssm.Parameter
	before := acctest.RandString(10)
	after := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterBasicConfig(before, "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.foo", &beforeParam),
				),
			},
			{
				Config: testAccAWSSSMParameterBasicConfig(after, "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.foo", &afterParam),
					testAccCheckAWSSSMParameterRecreated(t, &beforeParam, &afterParam),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_secure(t *testing.T) {
	var param ssm.Parameter
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterSecureConfig(name, "secret"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.secret_foo", &param),
					resource.TestCheckResourceAttr("aws_ssm_parameter.secret_foo", "value", "secret"),
					resource.TestCheckResourceAttr("aws_ssm_parameter.secret_foo", "type", "SecureString"),
				),
			},
		},
	})
}

func TestAccAWSSSMParameter_secure_with_key(t *testing.T) {
	var param ssm.Parameter
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMParameterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMParameterSecureConfigWithKey(name, "secret"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMParameterExists("aws_ssm_parameter.secret_foo", &param),
					resource.TestCheckResourceAttr("aws_ssm_parameter.secret_foo", "value", "secret"),
					resource.TestCheckResourceAttr("aws_ssm_parameter.secret_foo", "type", "SecureString"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMParameterRecreated(t *testing.T,
	before, after *ssm.Parameter) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.Name == *after.Name {
			t.Fatalf("Expected change of SSM Param Names, but both were %v", *before.Name)
		}
		return nil
	}
}

func testAccCheckAWSSSMParameterExists(n string, param *ssm.Parameter) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Parameter ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		paramInput := &ssm.GetParametersInput{
			Names: []*string{
				aws.String(rs.Primary.Attributes["name"]),
			},
			WithDecryption: aws.Bool(true),
		}

		resp, err := conn.GetParameters(paramInput)
		if err != nil {
			return err
		}

		if len(resp.Parameters) == 0 {
			return fmt.Errorf("Expected AWS SSM Parameter to be created, but wasn't found")
		}

		*param = *resp.Parameters[0]

		return nil
	}
}

func testAccCheckAWSSSMParameterDisappears(param *ssm.Parameter) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		paramInput := &ssm.DeleteParameterInput{
			Name: param.Name,
		}

		_, err := conn.DeleteParameter(paramInput)
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckAWSSSMParameterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_parameter" {
			continue
		}

		paramInput := &ssm.GetParametersInput{
			Names: []*string{
				aws.String(rs.Primary.Attributes["name"]),
			},
		}

		resp, _ := conn.GetParameters(paramInput)

		if len(resp.Parameters) > 0 {
			return fmt.Errorf("Expected AWS SSM Parameter to be gone, but was still found")
		}

		return nil
	}

	return nil
}

func testAccAWSSSMParameterBasicConfig(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "foo" {
  name  = "test_parameter-%s"
  type  = "String"
  value = "%s"
}
`, rName, value)
}

func testAccAWSSSMParameterBasicConfigOverwrite(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "foo" {
  name  = "test_parameter-%s"
  type  = "String"
  value = "%s"
  overwrite = true
}
`, rName, value)
}

func testAccAWSSSMParameterSecureConfig(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "secret_foo" {
  name  = "test_secure_parameter-%s"
  type  = "SecureString"
  value = "%s"
}
`, rName, value)
}

func testAccAWSSSMParameterSecureConfigWithKey(rName string, value string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "secret_foo" {
  name  = "test_secure_parameter-%s"
  type  = "SecureString"
  value = "%s"
	key_id = "${aws_kms_key.test_key.id}"
}

resource "aws_kms_key" "test_key" {
  description             = "KMS key 1"
  deletion_window_in_days = 7
}
`, rName, value)
}
