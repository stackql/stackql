package provider

func getProviderMap(providerName string) map[string]interface{} {
	googleMap := map[string]interface{}{
		"name": providerName,
	}
	return googleMap
}

func getProviderMapExtended(providerName string) map[string]interface{} {
	return getProviderMap(providerName)
}
