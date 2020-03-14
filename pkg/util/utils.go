package util

import (
	"fmt"
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"os"
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

// GetIngressHostNameFor generates a host name for the Ingress host
func GetIngressHostNameFor(atlasMap *v1alpha1.AtlasMap) string {
	hostName := atlasMap.Spec.RouteHostName
	if len(hostName) == 0 {
		hostName = fmt.Sprintf("%s-%s", atlasMap.Name, atlasMap.Namespace)
	}
	return hostName
}

// ImageName generates a container image name from the given name and tag
func ImageName(image string, tag string) string {
	return fmt.Sprintf("%s:%s", image, tag)
}

// GetEnvVar gets the value of the given environment variable or returns a default value if it does not exist
func GetEnvVar(name string, defaultValue string) string {
	value, exists := os.LookupEnv(name)
	if exists {
		return value
	}
	return defaultValue
}
