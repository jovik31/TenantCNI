package v1alpha1

import(
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Tenant is a specification for a Tenant resource
type Tenant struct{

	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TenantSpec `json:"spec"`
}

type TenantSpec struct{

	Name string `json:"name"`
	Nodes []Node `json:"nodes"`
}

type Node struct{	
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// TenantList is a list of Tenant resources
type TenantList struct{	
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta`json:"metadata,omitempty"`

	Items []Tenant `json:"items"`
}



