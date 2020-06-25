package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ZooKeeperClientPort         int32  = 2181
	ZooKeeperServerPort         int32  = 2888
	ZooKeeperLeaderElectionPort int32  = 3888
	ZooKeeperImage              string = "riet/kubernetes-zookeeper:1.0-3.4.10"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZookeeperSpec defines the desired state of Zookeeper
type ZookeeperSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Servers      int32                       `json:"servers"`
	Resources    corev1.ResourceRequirements `json:"resources,omitempty"`
	StorageClass string                      `json:"storageClass,omitempty"`
	StorageSize  string                      `json:"storageSize,omitempty"`
	Heap         string                      `json:"heap"`
}

// ZookeeperStatus defines the observed state of Zookeeper
type ZookeeperStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Zookeeper is the Schema for the zookeepers API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=zookeepers,scope=Namespaced
type Zookeeper struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZookeeperSpec   `json:"spec,omitempty"`
	Status ZookeeperStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ZookeeperList contains a list of Zookeeper
type ZookeeperList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Zookeeper `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Zookeeper{}, &ZookeeperList{})
}
