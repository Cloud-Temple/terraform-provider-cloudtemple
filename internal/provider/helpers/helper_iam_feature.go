package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// maxFeatureSubNesting est le nombre de niveaux d'imbrication "subfeatures" que
// le schéma de data_source_iam_features déclare (features -> subfeatures ->
// subfeatures) : 2 arêtes "subfeatures", donc 3 niveaux de nœuds. Le niveau le
// plus profond ne déclare PAS "subfeatures" : y émettre la clé — même une liste
// vide n'est tolérée par le writer SDK que tant qu'elle reste vide, une liste
// non vide casse — déclenche "Invalid address to set" (classe #243) et rend
// TOUTE la datasource inutilisable.
//
// La profondeur réelle du catalogue de features est 2 (un seul niveau de
// sous-features, observé sur GET /iam/v2/features, IAM API v2.47.0) ; cette
// borne garde donc une marge. Si l'API renvoyait un arbre plus profond, les
// sous-features au-delà sont TRONQUÉES (la datasource reste lisible) au lieu de
// provoquer un crash, et dataSourceFeaturesRead émet alors un warning. NB : le
// swagger IAM est incomplet (il ne documente pas "subFeatures", pourtant
// renvoyé par l'API).
//
// Cette borne et la profondeur déclarée par le schéma DOIVENT rester cohérentes :
// toute modification de l'une exige la mise à jour de l'autre.
const maxFeatureSubNesting = 2

// FlattenFeature convertit un objet Feature en une map compatible avec le schéma
// Terraform de la datasource cloudtemple_iam_features. Un feature nil (élément
// null renvoyé par l'API) donne nil plutôt que de paniquer ; les appelants
// (dataSourceFeaturesRead, récursion interne) ignorent déjà les nil en amont.
func FlattenFeature(feature *client.Feature) map[string]interface{} {
	if feature == nil {
		return nil
	}
	return flattenFeature(feature, maxFeatureSubNesting)
}

// flattenFeature aplatit récursivement un Feature en respectant strictement la
// profondeur d'imbrication "subfeatures" déclarée par le schéma. remaining est
// le nombre de niveaux "subfeatures" encore déclarés sous le nœud courant.
func flattenFeature(feature *client.Feature, remaining int) map[string]interface{} {
	result := map[string]interface{}{
		"id":   feature.ID,
		"name": feature.Name,
	}

	// On n'émet "subfeatures" que tant que le schéma le déclare à ce niveau, en
	// conservant une liste vide lorsqu'il n'y a pas d'enfant (forme du state
	// inchangée par rapport à l'historique). À remaining == 0 (niveau le plus
	// profond), on n'émet pas la clé : d'éventuels enfants y sont tronqués plutôt
	// que de casser la lecture de la datasource.
	if remaining > 0 {
		subfeatures := make([]map[string]interface{}, 0, len(feature.SubFeatures))
		for _, subfeature := range feature.SubFeatures {
			if subfeature == nil {
				continue // un élément null ne doit pas faire paniquer le flatten
			}
			subfeatures = append(subfeatures, flattenFeature(subfeature, remaining-1))
		}
		result["subfeatures"] = subfeatures
	}

	return result
}

// FeatureExceedsDeclaredDepth indique si l'arbre de sous-features enraciné en
// feature est plus profond que ce que le schéma de la datasource peut
// représenter (maxFeatureSubNesting niveaux d'imbrication "subfeatures"). Au-delà,
// FlattenFeature tronque silencieusement ; le Read s'appuie sur ce détecteur
// déterministe pour émettre un warning de troncature.
func FeatureExceedsDeclaredDepth(feature *client.Feature) bool {
	return featureDepthExceeds(feature, maxFeatureSubNesting)
}

// featureDepthExceeds renvoie true dès qu'une sous-feature RÉELLE existe à un
// niveau où le schéma ne déclare plus "subfeatures" (remaining == 0). Les
// éléments nil sont ignorés, exactement comme FlattenFeature les ignore : le
// détecteur de troncature reflète ainsi précisément ce que le flatten émet.
func featureDepthExceeds(feature *client.Feature, remaining int) bool {
	if feature == nil {
		return false
	}
	for _, subfeature := range feature.SubFeatures {
		if subfeature == nil {
			continue // ignoré par le flatten : ne compte pas comme profondeur
		}
		if remaining <= 0 {
			return true // une sous-feature réelle existe à un niveau non représentable
		}
		if featureDepthExceeds(subfeature, remaining-1) {
			return true
		}
	}
	return false
}
