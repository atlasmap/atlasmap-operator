package util

import (
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	"github.com/magiconair/properties/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
)

func TestGetIngressHostNameFor(t *testing.T) {
	atlasMap := &v1alpha1.AtlasMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
	}

	hostName := GetIngressHostNameFor(atlasMap)
	assert.Equal(t, hostName, "test-name-test-namespace")
}

func TestImageName(t *testing.T) {
	image := ImageName("docker.io/test/image", "1.2.3")
	assert.Equal(t, image, "docker.io/test/image:1.2.3")
}

func TestGetEnvVar(t *testing.T) {
	varName := "TEST_VAR"
	varValue := "test value"
	varDefaultValue := varValue + " default"

	os.Setenv(varName, varValue)
	enrVar := GetEnvVar(varName, varDefaultValue)
	assert.Equal(t, enrVar, varValue)

	os.Unsetenv(varName)

	enrVar = GetEnvVar(varName, varDefaultValue)
	assert.Equal(t, enrVar, varDefaultValue)
}