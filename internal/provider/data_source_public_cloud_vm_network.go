package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a Public Cloud VM Instances network from the catalogue, by `id` or by `name`. The resolved network `id` is the value a VM network adapter attaches to via its `network_id`.",

		ReadContext: publicCloudVMNetworkRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the network to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the network to retrieve. Conflicts with `id`. Network names are not guaranteed unique — if several networks share the name the lookup fails and you must use `id`.",
			},
		},
	}
}

func publicCloudVMNetworkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var network *client.PublicCloudVMNetwork

	if name := d.Get("name").(string); name != "" {
		// No server-side name filter on /networks: list the catalogue and match by
		// exact name. Network names are NOT guaranteed unique, so refuse an
		// ambiguous match rather than pick one arbitrarily.
		networks, err := c.PublicCloudVM().Network().List(ctx)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find network named %q: %s", name, err))
		}
		var matches []*client.PublicCloudVMNetwork
		for _, n := range networks {
			if n.Name == name {
				matches = append(matches, n)
			}
		}
		switch len(matches) {
		case 0:
			return diag.FromErr(fmt.Errorf("failed to find network named %q", name))
		case 1:
			network = matches[0]
		default:
			ids := make([]string, len(matches))
			for i, m := range matches {
				ids[i] = m.ID
			}
			return diag.FromErr(fmt.Errorf("found %d networks named %q (ids: %s); use id to disambiguate", len(matches), name, strings.Join(ids, ", ")))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		var err error
		network, err = c.PublicCloudVM().Network().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if network == nil {
			return diag.FromErr(fmt.Errorf("failed to find network with id %q", id))
		}
	}

	d.SetId(network.ID)
	for k, v := range helpers.FlattenPublicCloudVMNetwork(network) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
