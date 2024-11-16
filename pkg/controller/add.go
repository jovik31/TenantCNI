package controller

import (
	"context"
	"log"

	"github.com/jovik31/tenant/pkg/k8s"
	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"
	"github.com/jovik31/tenant/pkg/network/routing"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

func (c *Controller) addTenant(key string) error {

	//Get the namespace and key from the queued object
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
		return err
	}

	//Get tenant object from the tenant Lister from the cache
	tenant, err := c.tenantLister.Tenants(namespace).Get(name)
	if err != nil {
		log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
	}
	//Get the newest tenant version
	/*tenant, err := c.refreshTenant(namespace, name)
	if err != nil {
		log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
		return err
	}*/
	//Every change we need to make to tenant is going to be done on a copy of the tenant

	//Get clientset to get current node name
	kubeSet, err := k8s.GetKubeClientSet()
	if err != nil {
		log.Print("Error getting kube client set: ", err.Error())
	}

	currentNodeName, err := k8s.GetCurrentNodeName(kubeSet)
	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
	}

	//Check if the current node is part of the node list for the tenant
	newTenant := tenant.DeepCopy()
	if existsNode(newTenant.Spec.Nodes, currentNodeName) {

		s, err := ipam.NewNodeStore(defaultNodeDir, currentNodeName)
		if err != nil {
			log.Print("Error creating node store: ", err.Error())
		}
		s.LoadNodeData()
		nim, err := ipam.NewNodeIPAM(s, currentNodeName)
		if err != nil {
			log.Print("Error creating node IPAM: ", err.Error())
		}
		//Allocate and configure the tenant files with the information necessary
		nim.AllocateTenant(newTenant.Spec.Name, newTenant.Spec.VNI, newTenant.Spec.Prefix)

		//After configuring all the tenant files we need to set the currentNode annotations to show that the tenant is enabled
		//And add the values for Vtep IP, Node IP and VtepMac Address on the tenant object
		t, err := ipam.NewTenantStore(defaultNodeDir, newTenant.Name)
		if err != nil {

			log.Println("Error creating tenant store", err.Error())
		}

		t.LoadTenantData()
		//Get access to the tenant information to register in the Tenant CIDR
		tim, err := ipam.NewTenantIPAM(t, newTenant.Name)
		if err != nil {
			log.Println("Error creating tenant IPAM", err.Error())
		}

		//Set new tenant vxlan information to publish on the K8s API
		for index, element := range newTenant.Spec.Nodes {

			if element.Name == currentNodeName {

				err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
					tenant, err := c.refreshTenant(namespace, name)
					if err != nil {
						log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
						return err
					}

					newTenant = tenant.DeepCopy()
					newTenant.Spec.Nodes[index].NodeIP = nim.NodeStore.Data.NodeIP
					newTenant.Spec.Nodes[index].VtepIp = tim.TenantStore.Data.Vxlan.VtepIP
					newTenant.Spec.Nodes[index].VtepMac = tim.TenantStore.Data.Vxlan.VtepMac

					//Try to update resource
					_, err = c.tenantClient.Jovik31V1alpha1().Tenants(namespace).Update(context.TODO(), newTenant, v1.UpdateOptions{FieldManager: "tenant-controller"})
					return err
				})
				if err != nil {
					log.Println("Failed to update tenant resource on API")
					return err
				}
			}

		}
		//Tenant is present in more than one node. We need to setup vxlan for inter-node communication
		if len(newTenant.Spec.Nodes) > 1 {
			vxlanDevice, err := backend.InitVxlanDevice(tim.TenantStore.Data.TenantCIDR, tim.TenantStore.Data.Vxlan.VtepName, tim.TenantStore.Data.Vxlan.VNI, tim.TenantStore.Data.Vxlan.VtepMac)
			if err != nil {
				log.Println("Failed to create vxlan device")
			}
			log.Println("Vxlan device created: ", vxlanDevice.Name)

		}
		//Retry the node annotation if it fails
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			log.Printf("Updating node")
			//Add node annotation to signal that a tenant is present on this node
			node, err := k8s.GetCurrentNode(kubeSet, currentNodeName)
			if err != nil {
				log.Printf("Failed to retrive node")
			}

			err = k8s.StoreTenantAnnotationNode(kubeSet, node, tenant.Name)
			return err
		})
		if err != nil {
			log.Println("Failed to update Node resource on API")
			return err
		}

		//Allow tenant traffic forwarding
		if err := routing.AllowForwardingTenant(tim.TenantStore.Data.TenantCIDR); err != nil {
			log.Println("Failed to allow tenant traffic forwarding")
		}

		//Block traffic between tenants on the same node
		//1st: Get all tenants present in the node using the nodeStore
		tenantList := nim.NodeStore.Data.TenantList

		for tenant, tenantCIDR := range tenantList {
			if tenant != tim.TenantName && tenant != "defaulttenant" {
				stenantCIDR := tenantCIDR.String()
				routing.BlockTenant2TenantTraffic(tim.TenantStore.Data.TenantCIDR, stenantCIDR)
				log.Printf("Blocked traffic between tenants: %s and %s", tim.TenantName, tenant)

			} else {
				continue
			}
		}
		c.recorder.Event(newTenant, corev1.EventTypeNormal, "Add", "Tenant has been created on node: "+currentNodeName)
		return nil
	}

	return nil

}
