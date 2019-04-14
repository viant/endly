package run

import (
	"google.golang.org/api/run/v1alpha1"
	"math/rand"
	"time"
)

const (
	typeConfigurationsReady = "ConfigurationsReady"
	typeRoutesReady         = "RoutesReady"
	typeReady               = "Ready"
	statusTrue              = "True"
)

func generateRandomASCII(length int) string {
	rand.Seed((time.Now().UTC().UnixNano()))
	var result = make([]byte, length)
	//97-122
	for i := 0; i < length; i++ {
		result[i] = byte(97 + int(rand.Int31n(122-97)))
	}
	return string(result)
}

func isServiceReady(conditions []*run.ServiceCondition) bool {
	if len(conditions) == 0 {
		return false
	}
	isConfigurationsReady := false
	isRoutesReady := false
	isReady := false
	for _, condition := range conditions {
		if condition.Type == typeConfigurationsReady {
			isConfigurationsReady = condition.Status == statusTrue
		}
		if condition.Type == typeRoutesReady {
			isRoutesReady = condition.Status == statusTrue
		}
		if condition.Type == typeReady {
			isReady = condition.Status == statusTrue
		}
	}

	return isReady && isConfigurationsReady && isRoutesReady
}
