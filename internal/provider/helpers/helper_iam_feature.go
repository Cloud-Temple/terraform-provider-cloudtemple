package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenFeature convertit un objet Feature en une map compatible avec le schéma Terraform
func FlattenFeature(feature *client.Feature) map[string]interface{} {
	result := map[string]interface{}{
		"id":   feature.ID,
		"name": feature.Name,
	}

	// Traiter les sous-features de manière récursive
	if len(feature.SubFeatures) > 0 {
		subfeatures := make([]map[string]interface{}, len(feature.SubFeatures))
		for i, subfeature := range feature.SubFeatures {
			subfeatures[i] = FlattenFeature(subfeature)
		}
		result["subfeatures"] = subfeatures
	} else {
		result["subfeatures"] = []map[string]interface{}{}
	}

	return result
}
