package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AtlasMapSpec defines the desired state of AtlasMap
// +k8s:openapi-gen=true
type AtlasMapSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	Replicas      int32  `json:"replicas,omitempty"`
	RouteHostName string `json:"routeHostName,omitempty"`
	Image         string `json:"image,omitempty"`
	RequestCPU    string `json:"requestCPU,omitempty"`
	RequestMemory string `json:"requestMemory,omitempty"`
	LimitCPU      string `json:"limitCPU,omitempty"`
	LimitMemory   string `json:"limitMemory,omitempty"`
}

// AtlasMapStatus defines the observed state of AtlasMap
// +k8s:openapi-gen=true
type AtlasMapStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	URL   string `json:"URL,omitempty"`
	Image string `json:"image,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AtlasMap is the Schema for the atlasmaps API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type AtlasMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AtlasMapSpec   `json:"spec,omitempty"`
	Status AtlasMapStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AtlasMapList contains a list of AtlasMap
type AtlasMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AtlasMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AtlasMap{}, &AtlasMapList{})
}
