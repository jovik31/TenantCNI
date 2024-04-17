package controller

import (
	"context"
	"log"

	"github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	"github.com/jovik31/tenant/pkg/network/ipam"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


type EventObject struct {

	eventType string
	key string
	newObj interface{}
	oldObj interface{}
}


func existsNode(nodeList []v1alpha1.Node, currentNodeName string) bool {

	for _, element := range nodeList {
		if currentNodeName == element.Name {
			log.Print("Node is part of tenant")

			return true
		}
	}
	return false

}

func (c *Controller) refreshTenant(namespace string, name string) (*v1alpha1.Tenant, error){

	tenant, err:=c.tenantLister.Tenants(namespace).Get(name)
	if err!=nil{
		return nil, err
	}
	return tenant, nil
}



func (c *Controller) UpdateTenantResource(tenant *v1alpha1.Tenant, tenantOnFile *ipam.TenantData, namespace string ) error {
	
	newTenant := tenant.DeepCopy()
	newTenant.ObjectMeta.Name = tenantOnFile.TenantName
	newTenant.Spec.VNI = tenantOnFile.Vxlan.VNI
	newTenant.Spec.Prefix = tenantOnFile.TenantPrefix

	_, err := c.tenantClient.Jovik31V1alpha1().Tenants(namespace).Update(context.TODO(), newTenant, metaV1.UpdateOptions{FieldManager: "tenant-operator"})
	if err!=nil{
		log.Println("Failed to update tenant", err)
		return err
	}

	return nil
}