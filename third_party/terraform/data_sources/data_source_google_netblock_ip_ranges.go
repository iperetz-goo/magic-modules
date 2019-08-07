package google

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"io/ioutil"
	"net/http"
	"strings"
)

func dataSourceGoogleNetblockIpRanges() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceGoogleNetblockIpRangesRead,

		Schema: map[string]*schema.Schema{
			"range_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "cloud-netblocks",
			},
			"cidr_blocks": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"cidr_blocks_ipv4": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"cidr_blocks_ipv6": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataSourceGoogleNetblockIpRangesRead(d *schema.ResourceData, meta interface{}) error {

	rt := d.Get("range_type").(string)
	CidrBlocks := make(map[string][]string)

	switch rt {
	// Dynamic ranges
	case "cloud-netblocks":
		// https://cloud.google.com/compute/docs/faq#where_can_i_find_product_name_short_ip_ranges
		const CLOUD_NETBLOCK_DNS = "_cloud-netblocks.googleusercontent.com"
		CidrBlocks, err := getCidrBlocks(CLOUD_NETBLOCK_DNS)

		if err != nil {
			return err
		}
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
		d.Set("cidr_blocks_ipv6", CidrBlocks["cidr_blocks_ipv6"])
	case "google-netblocks":
		// https://support.google.com/a/answer/33786?hl=en
		const GOOGLE_NETBLOCK_DNS = "_spf.google.com"
		CidrBlocks, err := getCidrBlocks(GOOGLE_NETBLOCK_DNS)

		if err != nil {
			return err
		}
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
		d.Set("cidr_blocks_ipv6", CidrBlocks["cidr_blocks_ipv6"])
	// Static ranges
	case "restricted-googleapis":
		// https://cloud.google.com/vpc/docs/configure-private-google-access-hybrid
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "199.36.153.4/30")
		CidrBlocks["cidr_blocks"] = CidrBlocks["cidr_blocks_ipv4"]
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
	case "dns-forwarders":
		// https://cloud.google.com/dns/zones/#creating-forwarding-zones
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "35.199.192.0/19")
		CidrBlocks["cidr_blocks"] = CidrBlocks["cidr_blocks_ipv4"]
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
	case "iap-forwarders":
		// https://cloud.google.com/iap/docs/using-tcp-forwarding
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "35.235.240.0/20")
		CidrBlocks["cidr_blocks"] = CidrBlocks["cidr_blocks_ipv4"]
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
	case "health-checkers":
		// https://cloud.google.com/load-balancing/docs/health-checks#fw-ruleh
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "35.191.0.0/16")
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "130.211.0.0/22")
		CidrBlocks["cidr_blocks"] = CidrBlocks["cidr_blocks_ipv4"]
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
	case "legacy-health-checkers":
		// https://cloud.google.com/load-balancing/docs/health-check#fw-netlbs
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "35.191.0.0/16")
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "209.85.152.0/22")
		CidrBlocks["cidr_blocks_ipv4"] = append(CidrBlocks["cidr_blocks_ipv4"], "209.85.204.0/22")
		CidrBlocks["cidr_blocks"] = CidrBlocks["cidr_blocks_ipv4"]
		d.Set("cidr_blocks", CidrBlocks["cidr_blocks"])
		d.Set("cidr_blocks_ipv4", CidrBlocks["cidr_blocks_ipv4"])
	default:
		return fmt.Errorf("Unknown range_type: %s", rt)
	}

	d.SetId("netblock-ip-ranges-" + rt)

	return nil
}

func netblock_request(name string) (string, error) {
	response, err := http.Get(fmt.Sprintf("https://dns.google.com/resolve?name=%s&type=TXT", name))

	if err != nil {
		return "", fmt.Errorf("Error from _cloud-netblocks: %s", err)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return "", fmt.Errorf("Error to retrieve the domains list: %s", err)
	}

	return string(body), nil
}

func getCidrBlocks(netblock string) (map[string][]string, error) {
	var dnsNetblockList []string
	cidrBlocks := make(map[string][]string)

	response, err := netblock_request(netblock)

	if err != nil {
		return nil, err
	}

	splitedResponse := strings.Split(response, " ")

	for _, sp := range splitedResponse {
		if strings.HasPrefix(sp, "include:") {
			dnsNetblock := strings.Replace(sp, "include:", "", 1)
			dnsNetblockList = append(dnsNetblockList, dnsNetblock)
		}
	}

	for len(dnsNetblockList) > 0 {

		dnsNetblock := dnsNetblockList[0]

		dnsNetblockList[0] = ""
		dnsNetblockList = dnsNetblockList[1:]

		response, err = netblock_request(dnsNetblock)

		if err != nil {
			return nil, err
		}

		splitedResponse = strings.Split(response, " ")

		for _, sp := range splitedResponse {
			if strings.HasPrefix(sp, "ip4") {
				cdrBlock := strings.Replace(sp, "ip4:", "", 1)
				cidrBlocks["cidr_blocks_ipv4"] = append(cidrBlocks["cidr_blocks_ipv4"], cdrBlock)
				cidrBlocks["cidr_blocks"] = append(cidrBlocks["cidr_blocks"], cdrBlock)

			} else if strings.HasPrefix(sp, "ip6") {
				cdrBlock := strings.Replace(sp, "ip6:", "", 1)
				cidrBlocks["cidr_blocks_ipv6"] = append(cidrBlocks["cidr_blocks_ipv6"], cdrBlock)
				cidrBlocks["cidr_blocks"] = append(cidrBlocks["cidr_blocks"], cdrBlock)

			} else if strings.HasPrefix(sp, "include:") {
				cidr_block := strings.Replace(sp, "include:", "", 1)
				dnsNetblockList = append(dnsNetblockList, cidr_block)
			}
		}
	}

	return cidrBlocks, nil
}
