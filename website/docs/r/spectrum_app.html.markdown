---
layout: "cloudflare"
page_title: "Cloudflare: cloudflare_spectrum_app"
sidebar_current: "docs-cloudflare-resource-load-balancer"
description: |-
  Provides a Cloudflare Sprectrum Application resource.
---

# cloudflare_spectrum_app

## Example Usage

```hcl
# Define a spectrum application proxies ssh traffic
resource "cloudflare_spectrum_app" "ssh_proxy" {
  protocol = "tcp/22"
  dns = {
    type = "CNAME"
    name = "ssh.example.com"
  }
  
  origin_direct = [
    "tcp://109.151.40.129:22"
  ]
}
```

Provides a Cloudflare Load Balancer resource.  You can extend the power of Cloudflare's DDoS, TLS, and IP Firewall to your other TCP-based services. This allows you to proxy tcp traffic over the Cloudflare network.

## Argument Reference
* `protocol`  - (Required) The port configuration at Cloudflare’s edge. i.e. `tcp/22`
* `dns` - (Required) The name and type of DNS record for the Spectrum application. Fields documented below.
* `origin_direct` (Optional) A list of destination addresses to the origin. i.e. `tcp://192.0.2.1:22`
* `origin_dns` (Optional) A destination dns addresses to the origin. Fields documented below
* `origin_port` (Optional) If using `origin_dns` this is a required attribute. Origin port to proxy traffice to i.e. `22`
* `tls` (Optional) If `on` Cloudflare will decrypt traffic for your application at the edge. Defaults to `off`
* `ip_firewall` (Optional) Enables the IP Firewall for this application. Defaults to `true`
* `proxy_protocol` (Optional) Enables Proxy Protocol v1 to the origin. Defaults to `false`

**dns**

* `type` (Requried) The type of DNS record associated with the application. Valid values: `CNAME`
* `name` (Required) The name of the DNS record associated with the application.i.e. `ssh.example.com`

**origin_dns**

* `name` - (Required) Fully qualified domain name of the origin i.e. origin-ssh.example.com
## Attributes Reference

The following attributes are exported:

* `id` - Unique identifier in the API for the spectrum application.
* `created_on` - The RFC3339 timestamp of when the spectrum application was created.
* `modified_on` - The RFC3339 timestamp of when the spectrum application was last modified.