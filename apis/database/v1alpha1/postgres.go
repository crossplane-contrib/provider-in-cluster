package v1alpha1

import (
	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresParameters define the desired state of an AWS IAM Role.
type PostgresParameters struct {

	// DatabaseSize is the size of the database in a valid Go notation
	// e.g., 1Gi
	DatabaseSize string `json:"databaseSize"`

	// MasterUsername is the name for the master user.
	// Constraints:
	//    * Required for PostgreSQL.
	//    * Must be 1 to 63 letters or numbers.
	//    * First character must be a letter.
	//    * Cannot be a reserved word for the chosen database engine.
	// +immutable
	// +optional
	MasterUsername *string `json:"masterUsername,omitempty"`

	// Database specifies the default database to be created with the image
	// +optional
	Database *string `json:"database,omitempty"`

	// StorageClass specifies the storage classed used for the PVC.
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	// MasterPasswordSecretRef references the secret that contains the password used
	// in the creation of this RDS instance. If no reference is given, a password
	// will be auto-generated.
	// +optional
	// +immutable
	MasterPasswordSecretRef *runtimev1alpha1.SecretKeySelector `json:"masterPasswordSecretRef,omitempty"`
}

// An PostgresSpec defines the desired state of an Postgres.
type PostgresSpec struct {
	runtimev1alpha1.ResourceSpec `json:",inline"`
	ForProvider                  PostgresParameters `json:"forProvider"`
}

// PostgresExternalStatus keeps the state for the external resource
type PostgresExternalStatus struct {
	// The status of the PVC for this Postgres database
	PVCStatus string `json:"pvcStatus"`
}

// An PostgresStatus represents the observed state of an Postgres.
type PostgresStatus struct {
	runtimev1alpha1.ResourceStatus `json:",inline"`
	AtProvider                     PostgresExternalStatus `json:"atProvider"`
}

// +kubebuilder:object:root=true

// An Postgres is a managed resource that represents an AWS IAM Role.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,aws}
type Postgres struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresSpec   `json:"spec"`
	Status PostgresStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresList contains a list of IAMRoles
type PostgresList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Postgres `json:"items"`
}
