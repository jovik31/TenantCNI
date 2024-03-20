package v1alpha1

import (
	"net"

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
	VNI int `json:"vni"`
	Prefix int `json:"prefix"`
	Nodes []Node `json:"nodes"`
}

type Node struct{	
	Name string `json:"name"`
	VtepMac string `json:"vtepMac,omitempty"` //VTEP MAC address is saved using string format due to the fact that it generates an error with the cache informer, create string to Mac address 
	VtepIp net.IP `json:"vtepIp,omitempty"`
	NodeIP net.IP `json:"nodeIP,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// TenantList is a list of Tenant resources
type TenantList struct{	
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta`json:"metadata,omitempty"`

	Items []Tenant `json:"items,omitempty"`
}



