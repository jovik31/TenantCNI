package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SchemeGroupVersion is group version used to register these objects
var (
	SchemeGroupVersion = schema.GroupVersion{
	Group: "jovik31.dev",
	Version: "v1alpha1",
}
)

var (
	SchemeBuilder runtime.SchemeBuilder
	AddToScheme = SchemeBuilder.AddToScheme
)


func init() {
	SchemeBuilder.Register(addKnownTypes)}


func addKnownTypes(scheme *runtime.Scheme) error{

	scheme.AddKnownTypes(SchemeGroupVersion, &Tenant{}, &TenantList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	
	return nil
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}