package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AtlasMapSpec defines the desired state of AtlasMap
// +k8s:openapi-gen=true
type AtlasMapSpec struct {
	// Replicas determines the desired number of running AtlasMap pods
	Replicas int32 `json:"replicas,omitempty"`
	// RouteHostName sets the host name to use on the Ingress or OpenShift Route
	RouteHostName string `json:"routeHostName,omitempty"`
	// Version sets the version of the container image used for AtlasMap
	Version string `json:"version,omitempty"`
	// The amount of CPU to request
	// +kubebuilder:validation:Pattern=[0-9]+m?$
	RequestCPU string `json:"requestCPU,omitempty"`
	// The amount of memory to request
	// +kubebuilder:validation:Pattern=[0-9]+([kKmMgGtTpPeE]i?)?$
	RequestMemory string `json:"requestMemory,omitempty"`
	// The amount of CPU to limit
	// +kubebuilder:validation:Pattern=[0-9]+m?$
	LimitCPU string `json:"limitCPU,omitempty"`
	// The amount of memory to request
	// +kubebuilder:validation:Pattern=[0-9]+([kKmMgGtTpPeE]i?)?$
	LimitMemory string `json:"limitMemory,omitempty"`
}

// AtlasMapStatus defines the observed state of AtlasMap
// +k8s:openapi-gen=true
type AtlasMapStatus struct {
	// The URL where AtlasMap can be accessed
	URL string `json:"URL,omitempty"`
	// The container image that AtlasMap is using
	Image string `json:"image,omitempty"`
	// The current phase that the AtlasMap resource is in
	Phase AtlasMapPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true

// AtlasMap is the Schema for the atlasmaps API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.labelSelector
// +kubebuilder:printcolumn:name="URL",description=AtlasMap URL,type=string,JSONPath=`.status.URL`
// +kubebuilder:printcolumn:name="Image",description=AtlasMap image,type=string,JSONPath=`.status.image`
// +kubebuilder:printcolumn:name="Phase",description=AtlasMap phase,type=string,JSONPath=`.status.phase`
type AtlasMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AtlasMapSpec   `json:"spec,omitempty"`
	Status AtlasMapStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AtlasMapList contains a list of AtlasMap
type AtlasMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AtlasMap `json:"items"`
}

// AtlasMapPhase --
type AtlasMapPhase string

const (
	// AtlasMapPhasePhaseUndeployed --
	AtlasMapPhasePhaseUndeployed AtlasMapPhase = "Undeployed"
	// AtlasMapPhasePhaseDeploying --
	AtlasMapPhasePhaseDeploying AtlasMapPhase = "Deploying"
	// AtlasMapPhasePhaseDeployed --
	AtlasMapPhasePhaseDeployed AtlasMapPhase = "Deployed"
)

func init() {
	SchemeBuilder.Register(&AtlasMap{}, &AtlasMapList{})
}
