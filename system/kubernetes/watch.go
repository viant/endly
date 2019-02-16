package core

import (
	"fmt"
	"github.com/viant/endly"
	"k8s.io/api/apps/v1"
	"k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const maxStatusWaitTimeInSec = 10

func getWatcher(context *endly.Context, watchRequest interface{}) (watch.Interface, error) {
	if watchRequest == nil {
		return nil, nil
	}
	var watchResponse interface{}
	if err := endly.RunWithoutLogging(context, watchRequest, &watchResponse); err != nil {
		return nil, err
	}
	if result, ok := watchResponse.(watch.Interface); ok {
		return result, nil
	}
	return nil, fmt.Errorf("unsupported type: %T\n", watchResponse)
}

func waitUntilReady(watcher watch.Interface) error {
	if watcher == nil {
		return nil
	}
	defer watcher.Stop()
	channel := watcher.ResultChan()

	for {
		select {
		case event := <-channel:
			switch event.Type {
			case watch.Added, watch.Modified:

				switch res := event.Object.(type) {
				case *v1.Deployment:
					if isDeploymentReady(res) {
						return nil
					}
				case *corev1.ReplicationController:
					if isReplicationControllerReady(res) {
						return nil
					}
				case *corev1.Pod:
					if isPodReady(res) {
						return nil
					}
				case *v1beta2.Deployment:
					if isV1beta2DeploymentReady(res) {
						return nil
					}
				default:
					return nil
				}
			}
		case <-time.After(maxStatusWaitTimeInSec * time.Second):
			return nil
		}
	}
}

func isPodReady(resource *corev1.Pod) bool {
	for _, cond := range resource.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isDeploymentReady(resource *v1.Deployment) bool {
	return resource.Status.ReadyReplicas > 0
}

func isV1beta2DeploymentReady(resource *v1beta2.Deployment) bool {
	return resource.Status.ReadyReplicas > 0
}

func isReplicationControllerReady(resource *corev1.ReplicationController) bool {
	return resource.Status.ReadyReplicas > 0
}
