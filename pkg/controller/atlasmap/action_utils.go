package atlasmap

import (
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/atlasmap/atlasmap-operator/pkg/config"
	"github.com/atlasmap/atlasmap-operator/pkg/util"
	"github.com/atlasmap/atlasmap-operator/version"
	"strconv"
	"strings"
)

const (
	springBoot1ProbeEndpointPath = "/management/health"
	springBoot2ProbeEndpointPath = "/actuator/health"
)

func atlasMapLabels(atlasMap *v1alpha1.AtlasMap) map[string]string {
	return map[string]string{
		"atlasmap.io/name": atlasMap.ObjectMeta.Name,
		"atlasmap.io/version": atlasMapVersion(atlasMap),
		"atlasmap.io/operator.version": version.Version,
	}
}

func atlasMapImage(atlasMap *v1alpha1.AtlasMap) string {
	if len(atlasMap.Spec.Version) == 0 {
		return config.DefaultConfiguration.GetAtlasMapImage()
	}
	return util.ImageName(config.DefaultConfiguration.AtlasMapImage, atlasMap.Spec.Version)
}

func atlasMapVersion(atlasMap *v1alpha1.AtlasMap) string {
	if len(atlasMap.Spec.Version) == 0 {
		return config.DefaultConfiguration.Version
	}
	return atlasMap.Spec.Version
}

func atlasMapProbePath(atlasMap *v1alpha1.AtlasMap) (string, error) {
	// Handle differences in Spring Boot actuator health endpoint path
	if atlasMap.Spec.Version != "" {
		versionParts := strings.Split(atlasMap.Spec.Version, ".")
		if len(versionParts) > 1 {
			major, err := strconv.Atoi(versionParts[0])
			if err != nil {
				return "", err
			}

			minor, err := strconv.Atoi(versionParts[1])
			if err != nil {
				return "", err
			}

			if major == 1 && minor < 43 {
				return springBoot1ProbeEndpointPath, nil
			}
		}
	}
	return springBoot2ProbeEndpointPath, nil
}
