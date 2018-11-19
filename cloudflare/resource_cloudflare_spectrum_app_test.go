package cloudflare

import (
	"errors"
	"fmt"
	"testing"

	"time"

	"os"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

var zoneID string

func TestAccCloudflareSpectrumApplication_Basic(t *testing.T) {
	// multiple instances of this config would conflict but we only use it once
	t.Parallel()
	testStartTime := time.Now().UTC()
	var spectrumApp cloudflare.SpectrumApplication
	zone := os.Getenv("CLOUDFLARE_DOMAIN")
	rnd := acctest.RandString(10)
	name := "cloudflare_load_balancer." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudflareSpectrumApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCloudflareSpectrumApplicationConfigBasic(getZoneID(zone), rnd),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudflareSpectrumApplicationExists(name, &spectrumApp),
					testAccCheckCloudflareSpectrumApplicationIDIsValid(name, zone),
					// dont check that specified values are set, this will be evident by lack of plan diff
					// some values will get empty values
					resource.TestCheckResourceAttr(name, "pop_pools.#", "0"),
					resource.TestCheckResourceAttr(name, "region_pools.#", "0"),
					// also expect api to generate some values
					testAccCheckCloudflareSpectrumApplicationDates(name, &spectrumApp, testStartTime),
					resource.TestCheckResourceAttr(name, "proxied", "false"), // default value
					resource.TestCheckResourceAttr(name, "ttl", "30"),
					resource.TestCheckResourceAttr(name, "steering_policy", ""),
				),
			},
		},
	})
}

/**
Any change to a spectrum application  results in a new resource
Although the API client contains a modify method, this always results in 405 status
*/
func TestAccCloudflareSpectrumApplication_Update(t *testing.T) {
	t.Parallel()
	var spectrumApp cloudflare.SpectrumApplication
	var initialId string
	zone := os.Getenv("CLOUDFLARE_DOMAIN")
	rnd := acctest.RandString(10)
	name := "cloudflare_load_balancer." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudflareSpectrumApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCloudflareSpectrumApplicationConfigBasic(zone, rnd),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudflareSpectrumApplicationExists(name, &spectrumApp),
					testAccCheckCloudflareSpectrumApplicationIDIsValid(name, zone),
				),
			},
			{
				PreConfig: func() {
					initialId = spectrumApp.ID
				},
				Config: testAccCheckCloudflareSpectrumApplicationConfigBasicUpdated(zone, rnd),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudflareSpectrumApplicationExists(name, &spectrumApp),
					testAccCheckCloudflareSpectrumApplicationIDIsValid(name, zone),
					func(state *terraform.State) error {
						if initialId != spectrumApp.ID {
							// want in place update
							return fmt.Errorf("spectrum application id is different after second config applied ( %s -> %s )",
								initialId, spectrumApp.ID)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccCloudflareSpectrumApplication_CreateAfterManualDestroy(t *testing.T) {
	t.Parallel()
	var spectrumApp cloudflare.SpectrumApplication
	var initialId string
	zone := os.Getenv("CLOUDFLARE_DOMAIN")
	rnd := acctest.RandString(10)
	name := "cloudflare_load_balancer." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudflareSpectrumApplicationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCloudflareSpectrumApplicationConfigBasic(zone, rnd),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudflareSpectrumApplicationExists(name, &spectrumApp),
					testAccCheckCloudflareSpectrumApplicationIDIsValid(name, zone),
					testAccManuallyDeleteSpectrumApplication(name, &spectrumApp, &initialId),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccCheckCloudflareSpectrumApplicationConfigBasic(zone, rnd),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudflareSpectrumApplicationExists(name, &spectrumApp),
					testAccCheckCloudflareSpectrumApplicationIDIsValid(name, zone),
					func(state *terraform.State) error {
						if initialId == spectrumApp.ID {
							return fmt.Errorf("spectrum application id is unchanged even after we thought we deleted it ( %s )",
								spectrumApp.ID)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckCloudflareSpectrumApplicationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*cloudflare.API)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudflare_load_balancer" {
			continue
		}

		_, err := client.SpectrumApplication(rs.Primary.Attributes["zone_id"], rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("spectrum application still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckCloudflareSpectrumApplicationExists(n string, spectrumApp *cloudflare.SpectrumApplication) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Load Balancer ID is set")
		}

		client := testAccProvider.Meta().(*cloudflare.API)
		foundSpectrumApplication, err := client.SpectrumApplication(rs.Primary.Attributes["zone_id"], rs.Primary.ID)
		if err != nil {
			return err
		}

		*spectrumApp = foundSpectrumApplication

		return nil
	}
}

func testAccCheckCloudflareSpectrumApplicationIDIsValid(n, expectedZone string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if len(rs.Primary.ID) != 32 {
			return fmt.Errorf("invalid id %q, should be a string of length 32", rs.Primary.ID)
		}

		if rs.Primary.Attributes["zone"] != expectedZone {
			return fmt.Errorf("zone attribute %q doesn't match the expected value %q", rs.Primary.Attributes["zone"], expectedZone)
		}

		if zoneId, ok := rs.Primary.Attributes["zone_id"]; !ok || len(zoneId) < 1 {
			return errors.New("zone_id is unset, should always be set with id")
		}
		return nil
	}
}

func testAccCheckCloudflareSpectrumApplicationDates(n string, spectrumApp *cloudflare.SpectrumApplication, testStartTime time.Time) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, _ := s.RootModule().Resources[n]

		for timeStampAttr, serverVal := range map[string]time.Time{"created_on": *spectrumApp.CreatedOn, "modified_on": *spectrumApp.ModifiedOn} {
			timeStamp, err := time.Parse(time.RFC3339Nano, rs.Primary.Attributes[timeStampAttr])
			if err != nil {
				return err
			}

			if timeStamp != serverVal {
				return fmt.Errorf("state value of %s: %s is different than server created value: %s",
					timeStampAttr, rs.Primary.Attributes[timeStampAttr], serverVal.Format(time.RFC3339Nano))
			}

			// check retrieved values are reasonable
			// note this could fail if local time is out of sync with server time
			if timeStamp.Before(testStartTime) {
				return fmt.Errorf("State value of %s: %s should be greater than test start time: %s",
					timeStampAttr, timeStamp.Format(time.RFC3339Nano), testStartTime.Format(time.RFC3339Nano))
			}
		}

		return nil
	}
}

func testAccManuallyDeleteSpectrumApplication(name string, spectrumApp *cloudflare.SpectrumApplication, initialId *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[name]
		client := testAccProvider.Meta().(*cloudflare.API)
		*initialId = spectrumApp.ID
		err := client.DeleteSpectrumApplication(rs.Primary.Attributes["zone_id"], rs.Primary.ID)
		if err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckCloudflareSpectrumApplicationConfigBasic(zoneID, ID string) string {
	return fmt.Sprintf(`
resource "cloudflare_spectrum_app" "%[2]s" {
  zone_id  = "%[1]s"
  protocol = "tcp/22"
  dns = {
	"type" = "CNAME"
	"name" = "ssh.example.com"
  }
  origin_direct = ["tcp://120.120.102.10:23"]
}`, zoneID, ID)
}

func testAccCheckCloudflareSpectrumApplicationConfigBasicUpdated(zoneID, ID string) string {
	return fmt.Sprintf(`
resource "cloudflare_spectrum_app" "%[2]s" {
  zone_id  = "%[1]s"
  protocol = "tcp/22"
  dns = {
	"type" = "CNAME"
	"name" = "ssh.example.com"
  }
  origin_direct = ["tcp://121.120.102.10:23"]
}`, zoneID, ID)
}