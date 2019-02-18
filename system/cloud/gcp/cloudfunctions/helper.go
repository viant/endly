package cloudfunctions

import (
	"strings"
)

const defaultRegion = "us-central1"
const parentLocationTemplate = "projects/${gcp.projectID}/locations/${gcp.region}"

func initFullyQualifiedName(name string) string {
	if strings.HasPrefix(name, "projects/") {
		return name
	}
	return parentLocationTemplate + "/functions/" + name
}

func initRegion(region string) string {
	if region != "" {
		return region
	}
	return defaultRegion
}
