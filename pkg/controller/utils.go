package controller

import (
	"fmt"
	"log"

	tenantType "github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	"github.com/jovik31/tenant/pkg/k8s"
)

func existsNode(nodeList []tenantType.Node) bool {

	kubeSet := k8s.GetKubeClientSet()
	currentNodeName, err := k8s.GetCurrentNodeName(kubeSet)

	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
	}

	for _, element := range nodeList {
		if currentNodeName == element.Name {
			fmt.Print("Node is part of tenant")

			return true
		}
	}
	return false

}
