package util

import (
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// IsOpenShift returns true if the platform cluster is OpenShift
func IsOpenShift(config *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	_, err = client.ServerResourcesForGroupVersion("route.openshift.io/v1")

	if err != nil && errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// IngressHostName generates a host name for the Ingress host
func IngressHostName(atlasMap *v1alpha1.AtlasMap) string {
	hostName := atlasMap.Spec.RouteHostName
	if len(hostName) == 0 {
		hostName = atlasMap.Name
	}
	return hostName
}
