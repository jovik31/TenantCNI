package controller

import(
	"log"
	"context"
	"k8s.io/client-go/rest"

	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apixv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func RegisterTenantCRD(config *rest.Config) {
	apixClient, err := apixv1client.NewForConfig(config)
	errExit("Failed to load apiextensions client", err)

	crds := apixClient.CustomResourceDefinitions()

	tenantCRD := &apixv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tenants.jovik31.dev",
			
		},
		Spec: apixv1.CustomResourceDefinitionSpec{
			Scope: apixv1.NamespaceScoped,
			Group:   "jovik31.dev",
			Names: apixv1.CustomResourceDefinitionNames{
				Kind:"Tenant",
				Singular:"tenant",
				Plural:"tenants",
				ShortNames: []string{"tnt"},
				},
				Versions: []apixv1.CustomResourceDefinitionVersion{{
					Name: "v1alpha1",
					Served: true,
					Storage: true,
					Schema: &apixv1.CustomResourceValidation{
						OpenAPIV3Schema: &apixv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apixv1.JSONSchemaProps{
									"spec":{
										Type: "object",
										Properties: map[string]apixv1.JSONSchemaProps{
											"name":{
												Type: "string",
											},
											"nodes":{
												Type: "array",
												Items: &apixv1.JSONSchemaPropsOrArray{ 
													
													Schema: &apixv1.JSONSchemaProps{ 
														Type: "object",
														
															Properties:map[string]apixv1.JSONSchemaProps{

																		"name":{
																			Type: "string",
																		},
																		"vtepMac":{
																			Type: "string",
																		},
																		"vtepIp":{	
																			Type: "string",
																		},
																		"nodeIP":{
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
	_, err = crds.Create(context.TODO(), tenantCRD,metav1.CreateOptions{FieldManager: "tenant-controller"})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Print("Tenant CRD already registered")
		} else {
			errExit("Failed to create Tenant CRD", err)
		}
	}
}


func errExit(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %#v", msg, err)
	}
}