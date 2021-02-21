package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	awspolicy "github.com/jen20/awspolicyequivalence"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceS3BucketPolicy_basic(t *testing.T) {
	bucketName := acctest.RandomWithPrefix("tf-test-bucket")
	//region := testAccGetRegion()
	//hostedZoneID, _ := HostedZoneIDForRegion(region)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDataSourceS3BucketPolicyConfig_basic(bucketName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketPolicyExists("data.aws_s3_bucket_policy.policy"),
					testAccCheckAWSS3BucketPolicyPolicyMatch("data.aws_s3_bucket_policy.policy", "policy", "aws_s3_bucket_policy.bucket", "policy"),
					//resource.TestCheckResourceAttr("data.aws_s3_bucket.bucket", "region", region),
					//testAccCheckS3BucketDomainName("data.aws_s3_bucket.bucket", "bucket_domain_name", bucketName),
					//resource.TestCheckResourceAttr("data.aws_s3_bucket.bucket", "bucket_regional_domain_name", testAccBucketRegionalDomainName(bucketName, region)),
					//resource.TestCheckResourceAttr("data.aws_s3_bucket.bucket", "hosted_zone_id", hostedZoneID),
					//resource.TestCheckNoResourceAttr("data.aws_s3_bucket.bucket", "website_endpoint"),
				),
			},
		},
	})
}

func testAccCheckAWSS3BucketPolicyExists(n string) resource.TestCheckFunc {
	return testAccCheckAWSS3BucketPolicyExistsWithProvider(n, func() *schema.Provider { return testAccProvider })
}

func testAccCheckAWSS3BucketPolicyPolicyMatch(resource1, attr1, resource2, attr2 string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resource1]
		if !ok {
			return fmt.Errorf("Not found: %s", resource1)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		policy1, ok := rs.Primary.Attributes[attr1]
		if !ok {
			return fmt.Errorf("Attribute %q not found for %q", attr1, resource1)
		}

		rs, ok = s.RootModule().Resources[resource2]
		if !ok {
			return fmt.Errorf("Not found: %s", resource2)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		policy2, ok := rs.Primary.Attributes[attr2]
		if !ok {
			return fmt.Errorf("Attribute %q not found for %q", attr2, resource2)
		}

		areEquivalent, err := awspolicy.PoliciesAreEquivalent(policy1, policy2)
		if err != nil {
			return fmt.Errorf("Comparing AWS Policies failed: %s", err)
		}

		if !areEquivalent {
			return fmt.Errorf("AWS policies differ.\npolicy1: %s\npolicy2: %s", policy1, policy2)
		}

		return nil
	}
}

func testAccCheckAWSS3BucketPolicyExistsWithProvider(n string, providerF func() *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		fmt.Println("s.RootModule().Resources:", s.RootModule().Resources)
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		provider := providerF()

		conn := provider.Meta().(*AWSClient).s3conn
		_, err := conn.GetBucketPolicy(&s3.GetBucketPolicyInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			if isAWSErr(err, s3.ErrCodeNoSuchBucket, "") {
				return fmt.Errorf("s3 bucket not found")
			}
			return err
		}
		return nil

	}
}

func testAccAWSDataSourceS3BucketPolicyConfig_basic(bucketName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
  bucket = "%s"

  tags = {
    TestName = "TestAccAWSS3BucketPolicy_basic"
  }
}

resource "aws_s3_bucket_policy" "bucket" {
  bucket = aws_s3_bucket.bucket.bucket
  policy = data.aws_iam_policy_document.policy.json
}

data "aws_iam_policy_document" "policy" {
  statement {
    effect = "Allow"

    actions = [
      "s3:*",
    ]

    resources = [
      aws_s3_bucket.bucket.arn,
      "${aws_s3_bucket.bucket.arn}/*",
    ]

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }
  }
}

data "aws_s3_bucket_policy" "policy" {
  bucket = aws_s3_bucket.bucket.bucket
}

`, bucketName)
}
