package v1alpha1

import (
	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperatorParameters contains the user defined values for an operator.
type OperatorParameters struct {
	// +immutable
	OperatorName string `json:"operatorName"`

	// +immutable
	CatalogSource string `json:"catalogSource"`

	// +immutable
	CatalogSourceNamespace string `json:"catalogSourceNamespace"`

	// +immutable
	Channel string `json:"channel"`
}

// An OperatorSpec defines the desired state of an Operator.
type OperatorSpec struct {
	runtimev1alpha1.ResourceSpec `json:",inline"`
	ForProvider                  OperatorParameters `json:"forProvider"`
}

// An OperatorStatus represents the observed state of an Operator.
type OperatorStatus struct {
	runtimev1alpha1.ResourceStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// An Operator is a managed resource that represents an OLM Operator.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aws}
type Operator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorSpec   `json:"spec"`
	Status OperatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OperatorList contains a list of Operators
type OperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Operator `json:"items"`
}
