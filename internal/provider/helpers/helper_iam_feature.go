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

	// Traiter les sous-features de manière récursive. La clé "subfeatures" n'est
	// émise que lorsqu'il en existe : le schéma déclare une profondeur
	// d'imbrication finie dont l'élément le plus profond ne déclare pas
	// "subfeatures". Émettre une liste vide sur une feuille à ce niveau
	// déclencherait un "Invalid address to set" et casserait la lecture de la
	// datasource (classe #243).
	if len(feature.SubFeatures) > 0 {
		subfeatures := make([]map[string]interface{}, len(feature.SubFeatures))
		for i, subfeature := range feature.SubFeatures {
			subfeatures[i] = FlattenFeature(subfeature)
		}
		result["subfeatures"] = subfeatures
	}

	return result
}
