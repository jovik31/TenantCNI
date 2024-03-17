package crd

import (
	"context"
	"log"

	v1alpha1 "github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	"k8s.io/client-go/rest"

	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apixv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RegisterTenantCRD(config *rest.Config) {
	apixClient, err := apixv1client.NewForConfig(config)
	errExit("Failed to load apiextensions client", err)

	crds := apixClient.CustomResourceDefinitions()

	tenantCRD := &apixv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tenants." + v1alpha1.SchemeGroupVersion.Group,
		},
		Spec: apixv1.CustomResourceDefinitionSpec{
			Scope: apixv1.NamespaceScoped,
			Group: v1alpha1.SchemeGroupVersion.Group,
			Names: apixv1.CustomResourceDefinitionNames{
				Kind:       "Tenant",
				Singular:   "tenant",
				Plural:     "tenants",
				ShortNames: []string{"tnt"},
			},
			Versions: []apixv1.CustomResourceDefinitionVersion{{
				Name:    v1alpha1.SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
				Schema: &apixv1.CustomResourceValidation{
					OpenAPIV3Schema: &apixv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apixv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apixv1.JSONSchemaProps{
									"name": {
										Type: "string",
									},
									"nodes": {
										Type: "array",
										Items: &apixv1.JSONSchemaPropsOrArray{

											Schema: &apixv1.JSONSchemaProps{
												Type: "object",

												Properties: map[string]apixv1.JSONSchemaProps{

													"name": {
														Type: "string",
													},
													"vtepMac": {
														Type: "string",
													},
													"vtepIp": {
														Type: "string",
													},
													"nodeIP": {
														Type: "string",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}},
		},
	}

	log.Print("Registering tenant CRD")
	_, err = crds.Create(context.TODO(), tenantCRD, metav1.CreateOptions{FieldManager: "tenant-controller"})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Print("Tenant CRD already registered")
		} else {
			errExit("Failed to create Tenant CRD", err)
		}
	}
}

// TO DO
func RegisterDefaultTenant(tenantClient *tenantClientset.Clientset) {

	//Check every node in cluster and add it to the nodes for default tenant

	nodes := []v1alpha1.Node{
		{Name: "kind-control-plane"},
		{Name: "kind-worker"},
	}

	tenant := &v1alpha1.Tenant{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.Group + v1alpha1.SchemeGroupVersion.Version,
			Kind:       "Tenant",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "defaulttenant",
		},
		Spec: v1alpha1.TenantSpec{
			Name:  "defaulttenant",
			Nodes: nodes,
		},
	}
	_, err := tenantClient.Jovik31V1alpha1().Tenants("default").Create(context.TODO(), tenant, metav1.CreateOptions{FieldManager: "tenant-controller"})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Print("Default tenant aldready exists")
		} else {
			errExit("Failed to create Tenant CRD", err)
		}

	}

	//need to get all node names from cluster
	// cretate default tenant in every node
	// for each specific node
}

func errExit(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %#v", msg, err)
	}
}
