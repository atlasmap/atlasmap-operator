package apis

import (
	"github.com/atlasmap/atlasmap-operator/pkg/apis/atlasmap/v1alpha1"
	consolev1 "github.com/openshift/api/console/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme, consolev1.Install)
}
