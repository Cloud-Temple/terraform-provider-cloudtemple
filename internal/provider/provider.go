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
					Type:     schema.TypeString,
					Optional: true,
					Default:  "pp-shiva.cloud-temple.com",
				},
				"scheme": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "https",
				},
				"client_id": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTEMPLE_CLIENT_ID", nil),
				},
				"secret_id": {
					Type:        schema.TypeString,
					Required:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTEMPLE_SECRET_ID", nil),
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"cloudtemple_iam_company":                dataSourceCompany(),
				"cloudtemple_iam_features":               dataSourceFeatures(),
				"cloudtemple_iam_personal_access_token":  dataSourcePersonalAccessToken(),
				"cloudtemple_iam_personal_access_tokens": dataSourcePersonalAccessTokens(),
				"cloudtemple_iam_role":                   dataSourceRole(),
				"cloudtemple_iam_roles":                  dataSourceRoles(),
				"cloudtemple_iam_tenants":                dataSourceTenants(),
				"cloudtemple_iam_user":                   dataSourceUser(),
				"cloudtemple_iam_users":                  dataSourceUsers(),
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

func newStateWriter(d *schema.ResourceData) *stateWriter {
	return &stateWriter{d: d}
}

func (sw *stateWriter) set(key string, value interface{}) {
	err := sw.d.Set(key, value)
	if err != nil {
		sw.diags = append(sw.diags, diag.Errorf("failed to  set '%s': %v", key, err)...)
	}
}
