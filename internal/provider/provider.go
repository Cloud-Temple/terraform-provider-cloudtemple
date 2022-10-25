package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"address": {
					Description: "The HTTP address to connect to the API. Defaults to `pp-shiva.cloud-temple.com`. Can also be specified with the environment variable `CLOUDTEMPLE_HTTP_ADDR`.",
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTEMPLE_HTTP_ADDR", "pp-shiva.cloud-temple.com"),
				},
				"scheme": {
					Description: "The URL scheme to used to connect to the API. Default to `https`. Can also be specified with the environment variable `CLOUDTEMPLE_HTTP_SCHEME`.",
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTEMPLE_HTTP_SCHEME", "https"),
				},
				"client_id": {
					Description: "The client ID to login to the API with. Can also be specified with the environment variable `CLOUDTEMPLE_CLIENT_ID`.",
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTEMPLE_CLIENT_ID", nil),
				},
				"secret_id": {
					Description: "The secret ID to login to the API with. Can also be specified with the environment variable `CLOUDTEMPLE_SECRET_ID`.",
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTEMPLE_SECRET_ID", nil),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"cloudtemple_compute_content_libraries":       dataSourceContentLibraries(),
				"cloudtemple_compute_content_library":         dataSourceContentLibrary(),
				"cloudtemple_compute_datastore_cluster":       dataSourceDatastoreCluster(),
				"cloudtemple_compute_datastore_clusters":      dataSourceDatastoreClusters(),
				"cloudtemple_compute_datastore":               dataSourceDatastore(),
				"cloudtemple_compute_datastores":              dataSourceDatastores(),
				"cloudtemple_compute_folder":                  dataSourceFolder(),
				"cloudtemple_compute_folders":                 dataSourceFolders(),
				"cloudtemple_compute_guest_operating_system":  dataSourceGuestOperatingSystem(),
				"cloudtemple_compute_guest_operating_systems": dataSourceGuestOperatingSystems(),
				"cloudtemple_compute_host_cluster":            dataSourceHostCluster(),
				"cloudtemple_compute_host_clusters":           dataSourceHostClusters(),
				"cloudtemple_compute_host":                    dataSourceHost(),
				"cloudtemple_compute_hosts":                   dataSourceHosts(),
				"cloudtemple_compute_network_adapter":         dataSourceNetworkAdapter(),
				"cloudtemple_compute_network_adapters":        dataSourceNetworkAdapters(),
				"cloudtemple_compute_network":                 dataSourceNetwork(),
				"cloudtemple_compute_networks":                dataSourceNetworks(),
				"cloudtemple_compute_resource_pool":           dataSourceResourcePool(),
				"cloudtemple_compute_resource_pools":          dataSourceResourcePools(),
				"cloudtemple_compute_snapshots":               dataSourceSnapshots(),
				"cloudtemple_compute_virtual_controllers":     dataSourceVirtualControllers(),
				"cloudtemple_compute_virtual_datacenter":      dataSourceVirtualDatacenter(),
				"cloudtemple_compute_virtual_datacenters":     dataSourceVirtualDatacenters(),
				"cloudtemple_compute_virtual_disk":            dataSourceVirtualDisk(),
				"cloudtemple_compute_virtual_disks":           dataSourceVirtualDisks(),
				"cloudtemple_compute_virtual_machine":         dataSourceVirtualMachine(),
				"cloudtemple_compute_virtual_machines":        dataSourceVirtualMachines(),
				"cloudtemple_compute_virtual_switch":          dataSourceVirtualSwitch(),
				"cloudtemple_compute_virtual_switchs":         dataSourceVirtualSwitchs(),
				"cloudtemple_compute_worker":                  dataSourceWorker(),
				"cloudtemple_compute_workers":                 dataSourceWorkers(),
				"cloudtemple_iam_company":                     dataSourceCompany(),
				"cloudtemple_iam_features":                    dataSourceFeatures(),
				"cloudtemple_iam_personal_access_token":       dataSourcePersonalAccessToken(),
				"cloudtemple_iam_personal_access_tokens":      dataSourcePersonalAccessTokens(),
				"cloudtemple_iam_role":                        dataSourceRole(),
				"cloudtemple_iam_roles":                       dataSourceRoles(),
				"cloudtemple_iam_tenants":                     dataSourceTenants(),
				"cloudtemple_iam_user":                        dataSourceUser(),
				"cloudtemple_iam_users":                       dataSourceUsers(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"cloudtemple_iam_personal_access_token": resourcePersonalAccessToken(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
		// Setup a User-Agent for your API client (replace the provider name for yours):
		// userAgent := p.UserAgent("terraform-provider-scaffolding", version)
		// TODO: myClient.UserAgent = userAgent

		config := client.DefaultConfig()

		config.ClientID = d.Get("client_id").(string)
		config.SecretID = d.Get("secret_id").(string)

		client, err := client.NewClient(config)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		// We check now  that we can login to return this user as soon as
		// to the user
		_, err = client.Token(ctx)
		if err != nil {
			return nil, diag.Errorf("failed to login: %v", err)
		}

		return client, nil
	}
}

func getClient(meta any) *client.Client {
	return meta.(*client.Client)
}

func getUserID(ctx context.Context, client *client.Client, d *schema.ResourceData) (string, diag.Diagnostics) {
	userID, ok := d.Get("user_id").(string)
	if ok && userID != "" {
		return userID, nil
	}

	l, err := client.Token(ctx)
	if err != nil {
		return "", diag.Errorf("failed to get token: %v", err)
	}

	return l.UserID(), nil
}

func getTenantID(ctx context.Context, client *client.Client, d *schema.ResourceData) (string, diag.Diagnostics) {
	userID, ok := d.Get("tenant_id").(string)
	if ok && userID != "" {
		return userID, nil
	}

	l, err := client.Token(ctx)
	if err != nil {
		return "", diag.Errorf("failed to get token: %v", err)
	}

	return l.TenantID(), nil
}

func getCompanyID(ctx context.Context, client *client.Client, d *schema.ResourceData) (string, diag.Diagnostics) {
	userID, ok := d.Get("tenant_id").(string)
	if ok && userID != "" {
		return userID, nil
	}

	l, err := client.Token(ctx)
	if err != nil {
		return "", diag.Errorf("failed to get token: %v", err)
	}

	return l.CompanyID(), nil
}

type stateWriter struct {
	d     *schema.ResourceData
	diags diag.Diagnostics
}

func newStateWriter(d *schema.ResourceData, id string) *stateWriter {
	d.SetId(id)
	return &stateWriter{d: d}
}

func (sw *stateWriter) set(key string, value interface{}) {
	err := sw.d.Set(key, value)
	if err != nil {
		sw.diags = append(sw.diags, diag.Errorf("failed to  set '%s': %v", key, err)...)
	}
}
