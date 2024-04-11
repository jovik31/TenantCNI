package controller

import (
	"log"
	"github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	
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
