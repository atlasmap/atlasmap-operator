package util

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/atlasmap/atlasmap-operator/api/v1alpha1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("util")

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

// GetClusterVersionSemVer gets the semantic version for the OpenShift cluster
func getClusterVersionSemVer(config *rest.Config) *semver.Version {
	configClient, err := configv1client.NewForConfig(config)
	ctx := context.TODO()

	if err != nil {
		log.Error(err, "Failed to create config client")
		return nil
	}

	var openShiftSemVer *semver.Version
	clusterVersion, err := configClient.
		ConfigV1().
		ClusterVersions().
		Get(ctx, "version", metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			// default to OpenShift 3 as ClusterVersion API was introduced in OpenShift 4
			openShiftSemVer, _ = semver.NewVersion("3")
		} else {
			log.Error(err, "Failed to get OpenShift cluster version")
			return nil
		}
	} else {
		//latest version from the history
		v := clusterVersion.Status.History[0].Version
		openShiftSemVer, err = semver.NewVersion(v)
		if err != nil {
			log.Error(err, "Failed to get OpenShift cluster version")
			return nil
		}
	}
	return openShiftSemVer
}

// IsOpenShift43Plus returns true if the cluster version is OpenShift >= 4.3
func IsOpenShift43Plus(config *rest.Config) bool {
	openShiftSemVer := getClusterVersionSemVer(config)
	if openShiftSemVer != nil {
		constraint43, _ := semver.NewConstraint(">= 4.3")
		return constraint43.Check(openShiftSemVer)
	}
	return false
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

// ConsoleLinkName generates a name for an OpenShift ConsoleLink
func ConsoleLinkName(atlasMap *v1alpha1.AtlasMap) string {
	return atlasMap.Name + "-" + atlasMap.Namespace
}

// ConsoleLinkText generates text to be displayed for an OpenShift ConsoleLink
func ConsoleLinkText(atlasMap *v1alpha1.AtlasMap) string {
	name := atlasMap.Name
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.TrimPrefix(name, "atlasmap")
	name = strings.TrimSuffix(name, "atlasmap")
	name = strings.Title(name)
	return "AtlasMap - " + strings.TrimSpace(name)
}

// GetEnvVar gets the value of the given environment variable or returns a default value if it does not exist
func GetEnvVar(name string, defaultValue string) string {
	value, exists := os.LookupEnv(name)
	if exists {
		return value
	}
	return defaultValue
}
