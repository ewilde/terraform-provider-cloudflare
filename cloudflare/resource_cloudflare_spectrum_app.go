package cloudflare

import (
	"fmt"
	"log"
	"strings"

	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/pkg/errors"
)

const (
	cloudflareOn = "on"
	cloudflareOff = "off"
)

func resourceCloudflareSpectrumApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudflareSpectrumAppCreate,
		Read:   resourceCloudflareSpectrumAppRead,
		Update: resourceCloudflareSpectrumAppUpdate,
		Delete: resourceCloudflareSpectrumAppDelete,
		Importer: &schema.ResourceImporter{
			State: resourceCloudflareSpectrumAppImport,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"zone_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"protocol": {
				Type:       schema.TypeString,
				Required:   true,
			},

			"dns": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"origin_direct": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"origin_dns": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"origin_port": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"tls": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  cloudflareOff,
				ValidateFunc: validation.StringInSlice([]string{
					cloudflareOn,
					cloudflareOff,
				}, false),
			},

			"ip_firewall": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"proxy_protocol": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"created_on": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"modified_on": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudflareSpectrumAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	newSpectrumApp := applicationFromResource(d)
	zoneID := d.Get("zone_id").(string)
	
	log.Printf("[INFO] Creating Cloudflare Spectrum Application from struct: %+v", newSpectrumApp)

	r, err := client.CreateSpectrumApplication(zoneID, newSpectrumApp)
	if err != nil {
		return errors.Wrap(err, "error creating spectrum application for zone")
	}

	if r.ID == "" {
		return fmt.Errorf("failed to find id in Create response; resource was empty")
	}

	d.SetId(r.ID)

	log.Printf("[INFO] Cloudflare Specturm Application ID: %s", d.Id())

	return resourceCloudflareSpectrumAppRead(d, meta)
}

func resourceCloudflareSpectrumAppUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)

	application := applicationFromResource(d)

	log.Printf("[INFO] Updating Cloudflare Load Balancer from struct: %+v", application)

	_, err := client.UpdateSpectrumApplication(zoneID, application.ID, application)
	if err != nil {
		return errors.Wrap(err, "error creating spectrum application for zone")
	}

	return resourceCloudflareSpectrumAppRead(d, meta)
}

func resourceCloudflareSpectrumAppRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)
	applicationID := d.Id()

	application, err := client.SpectrumApplication(zoneID, applicationID)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP status 404") {
			log.Printf("[INFO] Spectrum application %s in zone %s not found", applicationID, zoneID)
			d.SetId("")
			return nil
		}
		return errors.Wrap(err,
			fmt.Sprintf("Error reading spectrum application resource from API for resource %s in zone %s", zoneID, applicationID))
	}

	d.Set("protocol", application.Protocol)

	if err := d.Set("dns", flattenDns(application.DNS)); err != nil {
		log.Printf("[WARN] Error setting dns on spectrum application %q: %s", d.Id(), err)
	}

	if err := d.Set("origin_direct", application.OriginDirect); err != nil {
		log.Printf("[WARN] Error setting origin direct on spectrum application %q: %s", d.Id(), err)
	}

	if err := d.Set("origin_dns", flattenOriginDns(application.OriginDNS)); err != nil {
		log.Printf("[WARN] Error setting origin dns on spectrum application %q: %s", d.Id(), err)
	}

	d.Set("origin_port", application.OriginPort)
	d.Set("tls", application.TLS)
	d.Set("ip_firewall", application.IPFirewall)
	d.Set("proxy_protocol", application.ProxyProtocol)

	d.Set("created_on", application.CreatedOn.Format(time.RFC3339Nano))
	d.Set("modified_on", application.ModifiedOn.Format(time.RFC3339Nano))

	return nil
}

func resourceCloudflareSpectrumAppDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	zoneID := d.Get("zone_id").(string)
	applicationID := d.Id()

	log.Printf("[INFO] Deleting Cloudflare Spectrum Application: %s in zone: %s", applicationID, zoneID)

	err := client.DeleteSpectrumApplication(zoneID, applicationID)
	if err != nil {
		return fmt.Errorf("error deleting Cloudflare Spectrum Application: %s", err)
	}

	return nil
}

func resourceCloudflareSpectrumAppImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*cloudflare.API)

	// split the id so we can lookup
	idAttr := strings.SplitN(d.Id(), "/", 2)
	var zoneName string
	var applicationID string
	if len(idAttr) == 2 {
		zoneName = idAttr[0]
		applicationID = idAttr[1]
	} else {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"zoneName/applicationID\"", d.Id())
	}
	zoneID, err := client.ZoneIDByName(zoneName)

	if err != nil {
		return nil, fmt.Errorf("error finding zoneName %q: %s", zoneName, err)
	}

	d.Set("zone_id", zoneID)
	d.SetId(applicationID)
	return []*schema.ResourceData{d}, nil
}

func expandDns(d interface{}) (cloudflare.SpectrumApplicationDNS) {
	cfg := d.(*schema.Set).List()
	dns := cloudflare.SpectrumApplicationDNS{}

	m := cfg[0].(map[string]interface{})
	dns.Type = m["type"].(string)
	dns.Name = m["name"].(string)

	return dns
}

func expandOriginDns(d interface{}) *cloudflare.SpectrumApplicationOriginDNS {
	cfg := d.(*schema.Set).List()
	dns := &cloudflare.SpectrumApplicationOriginDNS{}

	m := cfg[0].(map[string]interface{})
	dns.Name = m["name"].(string)

	return dns
}

func flattenDns(dns cloudflare.SpectrumApplicationDNS) []map[string]interface{} {
	flattened := map[string]interface{}{}
	flattened["type"] = dns.Type
	flattened["name"] = dns.Name

	return []map[string]interface{}{flattened}
}

func flattenOriginDns(dns *cloudflare.SpectrumApplicationOriginDNS) []map[string]interface{} {
	flattened := map[string]interface{}{}
	flattened["name"] = dns.Name

	return []map[string]interface{}{flattened}
}

func applicationFromResource(d *schema.ResourceData) cloudflare.SpectrumApplication {
	application := cloudflare.SpectrumApplication{
		Protocol:  d.Get("protocol").(string),
		DNS:       expandDns(d.Get("dns")),
	}

	if originDirect, ok := d.GetOk("origin_direct"); ok {
		application.OriginDirect = expandInterfaceToStringList(originDirect.(string))
	}

	if originDns, ok := d.GetOk("origin_dns"); ok {
		application.OriginDNS = expandOriginDns(originDns)
	}

	if originPort, ok := d.GetOk("origin_port"); ok {
		application.OriginPort = originPort.(int)
	}

	if tls, ok := d.GetOk("tls"); ok {
		application.TLS = tls.(string)
	}

	if ipFirewall, ok := d.GetOk("ip_firewall"); ok {
		application.IPFirewall = ipFirewall.(bool)
	}

	if proxyProtocol, ok := d.GetOk("proxy_protocol"); ok {
		application.ProxyProtocol = proxyProtocol.(bool)
	}

	return application
}