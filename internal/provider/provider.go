package provider

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

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
				"cloudtemple_activities":                      documentDatasource(dataSourceActivities(), "activity_read"),
				"cloudtemple_activity":                        documentDatasource(dataSourceActivity(), "activity_read"),
				"cloudtemple_backup_job_sessions":             documentDatasource(dataSourceBackupJobSessions(), "backup_read"),
				"cloudtemple_backup_job":                      documentDatasource(dataSourceBackupJob(), "backup_read"),
				"cloudtemple_backup_jobs":                     documentDatasource(dataSourceBackupJobs(), "backup_read"),
				"cloudtemple_backup_metrics":                  documentDatasource(dataSourceBackupMetrics(), "backup_read"),
				"cloudtemple_backup_sites":                    documentDatasource(dataSourceBackupSites(), "backup_read"),
				"cloudtemple_backup_sla_policies":             documentDatasource(dataSourceBackupSLAPolicies(), "backup_read"),
				"cloudtemple_backup_sla_policy":               documentDatasource(dataSourceBackupSLAPolicy(), "backup_read"),
				"cloudtemple_backup_spp_server":               documentDatasource(dataSourceBackupSPPServer(), "backup_read"),
				"cloudtemple_backup_spp_servers":              documentDatasource(dataSourceBackupSPPServers(), "backup_read"),
				"cloudtemple_backup_storages":                 documentDatasource(dataSourceBackupStorages(), "backup_read"),
				"cloudtemple_backup_vcenters":                 documentDatasource(dataSourceBackupVCenters(), "backup_read"),
				"cloudtemple_compute_content_libraries":       documentDatasource(dataSourceContentLibraries(), "compute_read"),
				"cloudtemple_compute_content_library":         documentDatasource(dataSourceContentLibrary(), "compute_read"),
				"cloudtemple_compute_datastore_cluster":       documentDatasource(dataSourceDatastoreCluster(), "compute_read"),
				"cloudtemple_compute_datastore_clusters":      documentDatasource(dataSourceDatastoreClusters(), "compute_read"),
				"cloudtemple_compute_datastore":               documentDatasource(dataSourceDatastore(), "compute_read"),
				"cloudtemple_compute_datastores":              documentDatasource(dataSourceDatastores(), "compute_read"),
				"cloudtemple_compute_folder":                  documentDatasource(dataSourceFolder(), "compute_read"),
				"cloudtemple_compute_folders":                 documentDatasource(dataSourceFolders(), "compute_read"),
				"cloudtemple_compute_guest_operating_system":  documentDatasource(dataSourceGuestOperatingSystem(), "compute_read"),
				"cloudtemple_compute_guest_operating_systems": documentDatasource(dataSourceGuestOperatingSystems(), "compute_read"),
				"cloudtemple_compute_host_cluster":            documentDatasource(dataSourceHostCluster(), "compute_read"),
				"cloudtemple_compute_host_clusters":           documentDatasource(dataSourceHostClusters(), "compute_read"),
				"cloudtemple_compute_host":                    documentDatasource(dataSourceHost(), "compute_read"),
				"cloudtemple_compute_hosts":                   documentDatasource(dataSourceHosts(), "compute_read"),
				"cloudtemple_compute_network_adapter":         documentDatasource(dataSourceNetworkAdapter(), "compute_read"),
				"cloudtemple_compute_network_adapters":        documentDatasource(dataSourceNetworkAdapters(), "compute_read"),
				"cloudtemple_compute_network":                 documentDatasource(dataSourceNetwork(), "compute_read"),
				"cloudtemple_compute_networks":                documentDatasource(dataSourceNetworks(), "compute_read"),
				"cloudtemple_compute_resource_pool":           documentDatasource(dataSourceResourcePool(), "compute_read"),
				"cloudtemple_compute_resource_pools":          documentDatasource(dataSourceResourcePools(), "compute_read"),
				"cloudtemple_compute_snapshots":               documentDatasource(dataSourceSnapshots(), "compute_read"),
				"cloudtemple_compute_virtual_controllers":     documentDatasource(dataSourceVirtualControllers(), "compute_read"),
				"cloudtemple_compute_virtual_datacenter":      documentDatasource(dataSourceVirtualDatacenter(), "compute_read"),
				"cloudtemple_compute_virtual_datacenters":     documentDatasource(dataSourceVirtualDatacenters(), "compute_read"),
				"cloudtemple_compute_virtual_disk":            documentDatasource(dataSourceVirtualDisk(), "compute_read"),
				"cloudtemple_compute_virtual_disks":           documentDatasource(dataSourceVirtualDisks(), "compute_read"),
				"cloudtemple_compute_virtual_machine":         documentDatasource(dataSourceVirtualMachine(), "compute_read"),
				"cloudtemple_compute_virtual_machines":        documentDatasource(dataSourceVirtualMachines(), "compute_read"),
				"cloudtemple_compute_virtual_switch":          documentDatasource(dataSourceVirtualSwitch(), "compute_read"),
				"cloudtemple_compute_virtual_switchs":         documentDatasource(dataSourceVirtualSwitchs(), "compute_read"),
				"cloudtemple_compute_worker":                  documentDatasource(dataSourceWorker(), "compute_read"),
				"cloudtemple_compute_workers":                 documentDatasource(dataSourceWorkers(), "compute_read"),
				"cloudtemple_iam_company":                     documentDatasource(dataSourceCompany(), "iam_read"),
				"cloudtemple_iam_features":                    documentDatasource(dataSourceFeatures(), "iam_read"),
				"cloudtemple_iam_personal_access_token":       documentDatasource(dataSourcePersonalAccessToken(), "iam_read"),
				"cloudtemple_iam_personal_access_tokens":      documentDatasource(dataSourcePersonalAccessTokens(), "iam_read"),
				"cloudtemple_iam_role":                        documentDatasource(dataSourceRole(), "iam_read"),
				"cloudtemple_iam_roles":                       documentDatasource(dataSourceRoles(), "iam_read"),
				"cloudtemple_iam_tenants":                     documentDatasource(dataSourceTenants(), "iam_read"),
				"cloudtemple_iam_user":                        documentDatasource(dataSourceUser(), "iam_read"),
				"cloudtemple_iam_users":                       documentDatasource(dataSourceUsers(), "iam_read"),
			},
			ResourcesMap: map[string]*schema.Resource{
				"cloudtemple_iam_personal_access_token": documentResource(resourcePersonalAccessToken(), "iam_offline_access"),
				"cloudtemple_compute_network_adapter":   documentResource(resourceNetworkAdapter(), "compute_write", "compute_read", "activity_read"),
				"cloudtemple_compute_virtual_disk":      documentResource(resourceVirtualDisk(), "compute_write", "compute_read", "activity_read"),
				"cloudtemple_compute_virtual_machine":   documentResource(resourceVirtualMachine(), "compute_write", "compute_read", "activity_read"),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
		config := client.DefaultConfig()

		config.ClientID = d.Get("client_id").(string)
		config.SecretID = d.Get("secret_id").(string)

		client, err := client.NewClient(config)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		userAgent := p.UserAgent("terraform-provider-cloudtemple", version)
		client.UserAgent = userAgent

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

func getUserID(ctx context.Context, client *client.Client, d *schema.ResourceData) (string, error) {
	userID, ok := d.Get("user_id").(string)
	if ok && userID != "" {
		return userID, nil
	}

	l, err := client.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	return l.UserID(), nil
}

func getTenantID(ctx context.Context, client *client.Client, d *schema.ResourceData) (string, error) {
	userID, ok := d.Get("tenant_id").(string)
	if ok && userID != "" {
		return userID, nil
	}

	l, err := client.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	return l.TenantID(), nil
}

func getCompanyID(ctx context.Context, client *client.Client, d *schema.ResourceData) (string, error) {
	userID, ok := d.Get("tenant_id").(string)
	if ok && userID != "" {
		return userID, nil
	}

	l, err := client.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %v", err)
	}

	return l.CompanyID(), nil
}

type stateWriter struct {
	d     *schema.ResourceData
	diags diag.Diagnostics
}

func newStateWriter(d *schema.ResourceData) *stateWriter {
	return &stateWriter{d: d}
}

func (sw *stateWriter) set(key string, value any) {
	if key == "id" {
		sw.d.SetId(value.(string))
		return
	}
	err := sw.d.Set(key, value)
	if err != nil {
		sw.diags = append(sw.diags, diag.Errorf("failed to  set '%s': %v", key, err)...)
	}
}

func (sw *stateWriter) save(obj any, skip []string) {
	skipFields := map[string]struct{}{}
	for _, s := range skip {
		skipFields[s] = struct{}{}
	}

	typ := reflect.TypeOf(obj)
	fields := map[string]reflect.Value{}

	switch typ.Kind() {
	case reflect.Map:
		for name, value := range obj.(map[string]interface{}) {
			fields[name] = reflect.ValueOf(value)
		}
	case reflect.Pointer:
		item := reflect.ValueOf(obj).Elem()
		if item.Kind() == reflect.Interface {
			item = item.Elem()
		}
		for _, field := range reflect.VisibleFields(item.Type()) {
			name, found := field.Tag.Lookup("terraform")
			if name == "-" {
				continue
			}
			if !found {
				sw.diags = append(sw.diags, diag.Errorf("no terraform tag found for %q", field.Name)...)
				continue
			}
			fields[name] = item.FieldByName(field.Name)
		}
	default:
		sw.diags = append(sw.diags, diag.Errorf("unexpected type %s", typ.String())...)
	}

	for name, value := range fields {
		if _, skip := skipFields[name]; skip {
			continue
		}

		converted := sw.convert(value, false, name, skipFields)
		sw.set(name, converted)
	}
}

func (sw *stateWriter) convert(v reflect.Value, alreadyInSlice bool, path string, skipFields map[string]struct{}) any {
	// Convert time.Time to its string representation
	if v.Type().String() == "time.Time" {
		return v.Interface().(time.Time).Format(time.RFC3339)
	}

	k := v.Kind()
	switch k {
	case reflect.Bool, reflect.Int, reflect.String:
		return v.Interface()

	case reflect.Slice:
		items := []interface{}{}
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			if item.Kind() == reflect.Ptr {
				item = item.Elem()
			}
			items = append(items, sw.convert(item, true, path+".#", skipFields))
		}
		return items

	case reflect.Struct:
		body := map[string]interface{}{}
		for _, field := range reflect.VisibleFields(v.Type()) {
			name, found := field.Tag.Lookup("terraform")

			p := path + "." + name
			if _, skip := skipFields[p]; skip || name == "-" {
				continue
			}

			if !found {
				sw.diags = append(sw.diags, diag.Errorf("no terraform tag found for %q", field.Name)...)
				continue
			}
			body[name] = sw.convert(v.FieldByName(field.Name), false, p, skipFields)
		}
		if alreadyInSlice {
			return body
		}
		return []interface{}{body}

	default:
		sw.diags = append(sw.diags, diag.Errorf("%s unknown kind %q", path, k.String())...)
		return nil
	}
}

// IsNumber is a ValidateFunc that ensures a string can be parsed as a number
func IsNumber(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return
	}

	if _, err := strconv.Atoi(v); err != nil {
		errors = append(errors, fmt.Errorf("expected %q to be a valid number, got %v", k, v))
	}

	return warnings, errors
}

func documentDatasource(r *schema.Resource, roles ...string) *schema.Resource {
	return documentPermissions("To query this datasource", r, roles...)
}

func documentResource(r *schema.Resource, roles ...string) *schema.Resource {
	return documentPermissions("To manage this resource", r, roles...)
}

func documentPermissions(prefix string, r *schema.Resource, roles ...string) *schema.Resource {
	if len(roles) == 1 {
		r.Description += fmt.Sprintf("\n%s you will need the `%s` role.", prefix, roles[0])

	} else {
		r.Description += fmt.Sprintf("\n%s you will need the following roles:\n", prefix)
		for i, r := range roles {
			roles[i] = fmt.Sprintf("  - `%s`", r)
		}
		r.Description += strings.Join(roles, "\n")
	}

	return r
}
