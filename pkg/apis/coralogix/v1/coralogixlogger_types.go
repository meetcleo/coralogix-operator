package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CoralogixLoggerSpec defines the desired state of CoralogixLogger
type CoralogixLoggerSpec struct {
	// Coralogix Private Key
	PrivateKey string `json:"private_key"`
}

// CoralogixLoggerStatus defines the observed state of CoralogixLogger
type CoralogixLoggerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CoralogixLogger is the Schema for the coralogixloggers API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=coralogixloggers,scope=Namespaced
type CoralogixLogger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoralogixLoggerSpec   `json:"spec,omitempty"`
	Status CoralogixLoggerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CoralogixLoggerList contains a list of CoralogixLogger
type CoralogixLoggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoralogixLogger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CoralogixLogger{}, &CoralogixLoggerList{})
}
