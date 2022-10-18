package provider

func interfaceSliceToStringSlice(l []interface{}) []string {
	slice := make([]string, len(l))

	for i, item := range l {
		slice[i] = item.(string)
	}

	return slice
}

func stringSliceToInterfaceSlice(l []string) []interface{} {
	slice := make([]interface{}, len(l))

	for i, item := range l {
		slice[i] = item
	}

	return slice
}
