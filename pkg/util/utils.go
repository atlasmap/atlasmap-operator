package util

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func GetClusterVersionSemVer(config *rest.Config) *semver.Version {
	configClient, err := configv1client.NewForConfig(config)

	var openShiftSemVer *semver.Version
	clusterVersion, err := configClient.
		ConfigV1().
		ClusterVersions().
		Get("version", metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			// default to OpenShift 3 as ClusterVersion API was introduced in OpenShift 4
			openShiftSemVer, _ = semver.NewVersion("3")
		} else {
			return nil
		}
	} else {
		//latest version from the history
		v := clusterVersion.Status.History[0].Version
		openShiftSemVer, err = semver.NewVersion(v)
		if err != nil {
			return nil
		}
	}
	return openShiftSemVer
}