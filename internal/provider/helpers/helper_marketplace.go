package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenMarketplaceItem converts a MarketplaceItem object to a map compatible with Terraform schema
func FlattenMarketplaceItem(item *client.MarketplaceItem) map[string]interface{} {
	// Flatten details
	details := []map[string]interface{}{
		{
			"overview":             item.Details.Overview,
			"how_to_use":           item.Details.HowToUse,
			"support":              item.Details.Support,
			"terms_and_conditions": item.Details.TermsAndConditions,
		},
	}

	// Flatten details_en
	detailsEN := []map[string]interface{}{
		{
			"overview":             item.DetailsEN.Overview,
			"how_to_use":           item.DetailsEN.HowToUse,
			"support":              item.DetailsEN.Support,
			"terms_and_conditions": item.DetailsEN.TermsAndConditions,
		},
	}

	// Flatten deployment options targets
	targets := []map[string]interface{}{}
	for _, target := range item.DeploymentOptions.Targets {
		targets = append(targets, map[string]interface{}{
			"key":   target.Key,
			"name":  target.Name,
			"skus":  target.SKUs,
			"files": target.Files,
		})
	}

	deploymentOptions := []map[string]interface{}{
		{
			"targets": targets,
		},
	}

	return map[string]interface{}{
		"id":                 item.ID,
		"name":               item.Name,
		"editor":             item.Editor,
		"icon":               item.Icon,
		"description":        item.Description,
		"description_en":     item.DescriptionEN,
		"creation_date":      item.CreationDate,
		"last_update":        item.LastUpdate,
		"categories":         item.Categories,
		"type":               item.Type,
		"version":            item.Version,
		"build":              item.Build,
		"details":            details,
		"details_en":         detailsEN,
		"deployment_options": deploymentOptions,
	}
}
