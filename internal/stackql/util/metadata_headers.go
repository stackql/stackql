package util

func GetServicesHeader(extended bool) []string {
	if extended {
		return []string{"id", "name", "title", "description", "version", "preferred"}
	}
	return []string{"id", "name", "title"}
}

func GetResourcesHeader(extended bool) []string {
	if extended {
		return []string{"name", "id", "description"}
	}
	return []string{"name", "id"}
}

func GetDescribeHeader(extended bool) []string {
	if extended {
		return []string{"name", "type", "description"}
	}
	return []string{"name", "type"}
}
