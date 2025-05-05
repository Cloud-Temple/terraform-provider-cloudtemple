package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func exists[T any](data []T, f func(T) bool) bool {
	for _, v := range data {
		if f(v) {
			return true
		}
	}

	return false
}

func GetStringList(d *schema.ResourceData, key string) []string {
	rawList, ok := d.Get(key).([]interface{})
	if !ok {
		return []string{}
	}

	stringList := make([]string, len(rawList))
	for i, v := range rawList {
		stringList[i] = v.(string)
	}
	return stringList
}
