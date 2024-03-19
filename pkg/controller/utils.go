package controller

import (
	"fmt"

	tenantType "github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
)

func existsNode(nodeList []tenantType.Node, currentNodeName string) bool {

	for _, element := range nodeList {
		if currentNodeName == element.Name {
			fmt.Print("Node is part of tenant")

			return true
		}
	}
	return false

}
