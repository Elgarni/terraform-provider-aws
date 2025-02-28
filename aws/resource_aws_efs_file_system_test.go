package aws

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/efs/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

func init() {
	resource.AddTestSweepers("aws_efs_file_system", &resource.Sweeper{
		Name: "aws_efs_file_system",
		F:    testSweepEfsFileSystems,
		Dependencies: []string{
			"aws_efs_mount_target",
			"aws_efs_access_point",
		},
	})
}

func testSweepEfsFileSystems(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).efsconn
	var sweeperErrs *multierror.Error

	input := &efs.DescribeFileSystemsInput{}
	err = conn.DescribeFileSystemsPages(input, func(page *efs.DescribeFileSystemsOutput, lastPage bool) bool {
		for _, filesystem := range page.FileSystems {
			id := aws.StringValue(filesystem.FileSystemId)

			log.Printf("[INFO] Deleting EFS File System: %s", id)

			r := resourceAwsEfsFileSystem()
			d := r.Data(nil)
			d.SetId(id)
			err := r.Delete(d, client)

			if err != nil {
				log.Printf("[ERROR] %s", err)
				sweeperErrs = multierror.Append(sweeperErrs, err)
				continue
			}
		}
		return true
	})
	if err != nil {
		sweeperErrs = multierror.Append(sweeperErrs, fmt.Errorf("error retrieving EFS File Systems: %w", err))
	}

	return sweeperErrs.ErrorOrNil()
}

func TestAccAWSEFSFileSystem_basic(t *testing.T) {
	var desc efs.FileSystemDescription
	resourceName := "aws_efs_file_system.test"
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					testAccMatchResourceAttrRegionalARN(resourceName, "arn", "elasticfilesystem", regexp.MustCompile(`file-system/fs-.+`)),
					testAccMatchResourceAttrRegionalHostname(resourceName, "dns_name", "efs", regexp.MustCompile(`fs-[^.]+`)),
					resource.TestCheckResourceAttr(resourceName, "performance_mode", "generalPurpose"),
					resource.TestCheckResourceAttr(resourceName, "creation_token", rName),
					resource.TestCheckResourceAttr(resourceName, "number_of_mount_targets", "0"),
					resource.TestCheckResourceAttr(resourceName, "encrypted", "false"),
					resource.TestCheckResourceAttr(resourceName, "throughput_mode", efs.ThroughputModeBursting),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_policy.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "size_in_bytes.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "size_in_bytes.0.value"),
					resource.TestCheckResourceAttrSet(resourceName, "size_in_bytes.0.value_in_ia"),
					resource.TestCheckResourceAttrSet(resourceName, "size_in_bytes.0.value_in_standard"),
					testAccMatchResourceAttrAccountID(resourceName, "owner_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSEFSFileSystemConfigWithPerformanceMode,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem("aws_efs_file_system.test2", &desc),
					resource.TestCheckResourceAttr("aws_efs_file_system.test2", "creation_token", "supercalifragilisticexpialidocious"),
					resource.TestCheckResourceAttr("aws_efs_file_system.test2", "performance_mode", "maxIO"),
				),
			},
		},
	})
}

func TestAccAWSEFSFileSystem_availabilityZoneName(t *testing.T) {
	var desc efs.FileSystemDescription
	resourceName := "aws_efs_file_system.test"
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfigAvailabilityZoneName(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttrPair(resourceName, "availability_zone_id", "data.aws_availability_zones.available", "zone_ids.0"),
					resource.TestCheckResourceAttrPair(resourceName, "availability_zone_name", "data.aws_availability_zones.available", "names.0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEFSFileSystem_tags(t *testing.T) {
	var desc efs.FileSystemDescription
	rName := acctest.RandomWithPrefix("tf-acc-tags")
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSEFSFileSystemConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSEFSFileSystemConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccAWSEFSFileSystemConfigWithMaxTags(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "50"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", rName),
					resource.TestCheckResourceAttr(resourceName, "tags.Another", "tag"),
					resource.TestCheckResourceAttr(resourceName, "tags.Tag45", "TestTagValue"),
				),
			},
		},
	})
}

func TestAccAWSEFSFileSystem_pagedTags(t *testing.T) {
	var desc efs.FileSystemDescription
	rInt := acctest.RandInt()
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfigPagedTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "10"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEFSFileSystem_kmsKey(t *testing.T) {
	var desc efs.FileSystemDescription
	rInt := acctest.RandInt()
	kmsKeyResourceName := "aws_kms_key.test"
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfigWithKmsKey(rInt, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttrPair(resourceName, "kms_key_id", kmsKeyResourceName, "arn"),
					resource.TestCheckResourceAttr(resourceName, "encrypted", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEFSFileSystem_kmsConfigurationWithoutEncryption(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAWSEFSFileSystemConfigWithKmsKey(rInt, false),
				ExpectError: regexp.MustCompile(`encrypted must be set to true when kms_key_id is specified`),
			},
		},
	})
}

func TestAccAWSEFSFileSystem_ProvisionedThroughputInMibps(t *testing.T) {
	var desc efs.FileSystemDescription
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfig_ProvisionedThroughputInMibps(1.0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "provisioned_throughput_in_mibps", "1"),
					resource.TestCheckResourceAttr(resourceName, "throughput_mode", efs.ThroughputModeProvisioned),
				),
			},
			{
				Config: testAccAWSEFSFileSystemConfig_ProvisionedThroughputInMibps(2.0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "provisioned_throughput_in_mibps", "2"),
					resource.TestCheckResourceAttr(resourceName, "throughput_mode", efs.ThroughputModeProvisioned),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEFSFileSystem_ThroughputMode(t *testing.T) {
	var desc efs.FileSystemDescription
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfig_ProvisionedThroughputInMibps(1.0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "provisioned_throughput_in_mibps", "1"),
					resource.TestCheckResourceAttr(resourceName, "throughput_mode", efs.ThroughputModeProvisioned),
				),
			},
			{
				Config: testAccAWSEFSFileSystemConfig_ThroughputMode(efs.ThroughputModeBursting),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "provisioned_throughput_in_mibps", "0"),
					resource.TestCheckResourceAttr(resourceName, "throughput_mode", efs.ThroughputModeBursting),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAWSEFSFileSystem_lifecyclePolicy(t *testing.T) {
	var desc efs.FileSystemDescription
	resourceName := "aws_efs_file_system.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfigWithLifecyclePolicy(
					"transition_to_ia",
					"invalid_value",
				),
				ExpectError: regexp.MustCompile(`got invalid_value`),
			},
			{
				Config: testAccAWSEFSFileSystemConfigWithLifecyclePolicy(
					"transition_to_ia",
					efs.TransitionToIARulesAfter30Days,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_policy.0.transition_to_ia", efs.TransitionToIARulesAfter30Days),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAWSEFSFileSystemConfigWithLifecyclePolicy(
					"transition_to_primary_storage_class",
					efs.TransitionToPrimaryStorageClassRulesAfter1Access,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_policy.0.transition_to_primary_storage_class", efs.TransitionToPrimaryStorageClassRulesAfter1Access),
				),
			},
			{
				Config: testAccAWSEFSFileSystemConfigRemovedLifecyclePolicy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					resource.TestCheckResourceAttr(resourceName, "lifecycle_policy.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSEFSFileSystem_disappears(t *testing.T) {
	var desc efs.FileSystemDescription
	resourceName := "aws_efs_file_system.test"
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ErrorCheck:   testAccErrorCheck(t, efs.EndpointsID),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSEFSFileSystemConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsFileSystem(resourceName, &desc),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsEfsFileSystem(), resourceName),
					testAccCheckResourceDisappears(testAccProvider, resourceAwsEfsFileSystem(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckEfsFileSystemDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).efsconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_efs_file_system" {
			continue
		}

		_, err := finder.FileSystemByID(conn, rs.Primary.ID)

		if tfresource.NotFound(err) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("EFS file system %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccCheckEfsFileSystem(resourceID string, fDesc *efs.FileSystemDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EFS file system ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).efsconn

		fs, err := finder.FileSystemByID(conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*fDesc = *fs

		return nil
	}
}

func testAccAWSEFSFileSystemConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %q
}
`, rName)
}

func testAccAWSEFSFileSystemConfigAvailabilityZoneName(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_efs_file_system" "test" {
  creation_token         = %q
  availability_zone_name = data.aws_availability_zones.available.names[0]
}
`, rName)
}

func testAccAWSEFSFileSystemConfigTags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %[1]q
  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccAWSEFSFileSystemConfigTags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  creation_token = %[1]q
  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}

func testAccAWSEFSFileSystemConfigPagedTags(rInt int) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  tags = {
    Name           = "test-efs-%[1]d"
    Another        = "tag"
    Test           = "yes"
    User           = "root"
    Page           = "1"
    Environment    = "prod"
    CostCenter     = "terraform"
    AcceptanceTest = "PagedTags"
    CreationToken  = "radek"
    PerfMode       = "max"
  }
}
`, rInt)
}

func testAccAWSEFSFileSystemConfigWithMaxTags(rName string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  tags = {
    Name    = %q
    Another = "tag"
    Tag00   = "TestTagValue"
    Tag01   = "TestTagValue"
    Tag02   = "TestTagValue"
    Tag03   = "TestTagValue"
    Tag04   = "TestTagValue"
    Tag05   = "TestTagValue"
    Tag06   = "TestTagValue"
    Tag07   = "TestTagValue"
    Tag08   = "TestTagValue"
    Tag09   = "TestTagValue"
    Tag10   = "TestTagValue"
    Tag11   = "TestTagValue"
    Tag12   = "TestTagValue"
    Tag13   = "TestTagValue"
    Tag14   = "TestTagValue"
    Tag15   = "TestTagValue"
    Tag16   = "TestTagValue"
    Tag17   = "TestTagValue"
    Tag18   = "TestTagValue"
    Tag19   = "TestTagValue"
    Tag20   = "TestTagValue"
    Tag21   = "TestTagValue"
    Tag22   = "TestTagValue"
    Tag23   = "TestTagValue"
    Tag24   = "TestTagValue"
    Tag25   = "TestTagValue"
    Tag26   = "TestTagValue"
    Tag27   = "TestTagValue"
    Tag28   = "TestTagValue"
    Tag29   = "TestTagValue"
    Tag30   = "TestTagValue"
    Tag31   = "TestTagValue"
    Tag32   = "TestTagValue"
    Tag33   = "TestTagValue"
    Tag34   = "TestTagValue"
    Tag35   = "TestTagValue"
    Tag36   = "TestTagValue"
    Tag37   = "TestTagValue"
    Tag38   = "TestTagValue"
    Tag39   = "TestTagValue"
    Tag40   = "TestTagValue"
    Tag41   = "TestTagValue"
    Tag42   = "TestTagValue"
    Tag43   = "TestTagValue"
    Tag44   = "TestTagValue"
    Tag45   = "TestTagValue"
    Tag46   = "TestTagValue"
    Tag47   = "TestTagValue"
  }
}
`, rName)
}

const testAccAWSEFSFileSystemConfigWithPerformanceMode = `
resource "aws_efs_file_system" "test2" {
  creation_token   = "supercalifragilisticexpialidocious"
  performance_mode = "maxIO"
}
`

func testAccAWSEFSFileSystemConfigWithKmsKey(rInt int, enable bool) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {
  description = "Terraform acc test %[1]d"
}

resource "aws_efs_file_system" "test" {
  encrypted  = %[2]t
  kms_key_id = aws_kms_key.test.arn
}
`, rInt, enable)
}

func testAccAWSEFSFileSystemConfig_ThroughputMode(throughputMode string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  throughput_mode = %q
}
`, throughputMode)
}

func testAccAWSEFSFileSystemConfig_ProvisionedThroughputInMibps(provisionedThroughputInMibps float64) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  provisioned_throughput_in_mibps = %f
  throughput_mode                 = "provisioned"
}
`, provisionedThroughputInMibps)
}

func testAccAWSEFSFileSystemConfigWithLifecyclePolicy(lpName string, lpVal string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "test" {
  lifecycle_policy {
    %s = %q
  }
}
`, lpName, lpVal)
}

const testAccAWSEFSFileSystemConfigRemovedLifecyclePolicy = `
resource "aws_efs_file_system" "test" {}
`
