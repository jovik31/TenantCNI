package controller

import (
	"context"
	"log"
	"slices"

	"github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	"github.com/jovik31/tenant/pkg/k8s"
	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Controller) deleteTenant(obj *EventObject) error {

	log.Print("Delete Tenant Received")
	deletedTenant := obj.oldObj.(*v1alpha1.Tenant)

	namespace, name, err := cache.SplitMetaNamespaceKey(obj.key)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
		return err
	}

	//Get the newest tenant version, and filter if this was a node deletion or a complete tenent deletion
	tenant, err := c.refreshTenant(namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Tenant not found: %s, deleted from K8s API ", deletedTenant.Name)
		} else {
			log.Printf("Failed with error: %s,", err.Error())
			return err
		}
	} else {
		log.Printf("Tenant not deleted from K8s API: %s", tenant.Name)
	}
	k8sClient, err := k8s.GetKubeClientSet()
	if err != nil {
		log.Print("Error getting kube client set: ", err.Error())
		return err
	}
	currentNodeName, err := k8s.GetCurrentNodeName(k8sClient)
	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
		return err
	}

	if existsNode(deletedTenant.Spec.Nodes, currentNodeName) {

		//First step is to delete all pods related to the specific tenant
		//Get pod list from the pod controller
		//Send delete signal to every pod

		podStore, err := ipam.NewPodStore()
		if err != nil {
			log.Printf("Error creating pod store: %s", err.Error())
		}

		podStore.LoadPodData()
		pim, err := ipam.NewPodIPAM(podStore)
		if err != nil {
			log.Printf("Error creating pod ipam: %s", err.Error())
		}
		podData := pim.PodStore.Data
		var deletePods []string
		for podName, tenantName := range podData.Pods {
			if tenantName == deletedTenant.Name {
				deletePods = append(deletePods, podName)

			}
		}
		podList, err := c.kubeClient.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to list pods with err %s", err.Error())
		}

		//Deletes all Pods related to the tenant
		for _, pod := range podList.Items {

			if slices.Contains(deletePods, pod.Name) {
				err = c.kubeClient.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, v1.DeleteOptions{})
				if err != nil {
					log.Printf("Failed to delete pod %s in namespace %s: %s", pod.Name, pod.Namespace, err.Error())
				}
			} else {
				continue
			}

		}

		//Get tenant fields to delete network devices
		t, err := ipam.NewTenantStore(defaultNodeDir, deletedTenant.Name)
		if err != nil {
			log.Println("Error creating tenant store", err.Error())
		}
		t.LoadTenantData()
		tim, err := ipam.NewTenantIPAM(t, deletedTenant.Name)
		if err != nil {
			log.Println("Error creating tenant IPAM", err.Error())
		}
		tim.TenantStore.LoadTenantData()
		//No need to delete routes. When devices are deleted, the routes are removed.
		tenantBridge := tim.TenantStore.Data.Bridge.Name
		vtepDevice := tim.TenantStore.Data.Vxlan.VtepName

		//Need to wait for all pods to be deleted from tenant before deleting devices and tenantStore
		//We reload tenant daata until we get all pods removed
		//log.Print(pim.PodStore.Data.Pods.)
		//for len()!=0 {
		//tim.TenantStore.LoadTenantData()
		//log.Print(tim.TenantStore.Data.IPs)

		//}
		log.Printf("All pods are deleted, proceed with node deletion process")
		//Delete all network devices from the tenant in the node
		if err := backend.DeleteTenantBridge(tenantBridge); err != nil {
			log.Printf("Error deleting bridge %s", err)
		}

		if err := backend.DeleteVxLANDevice(vtepDevice); err != nil {
			log.Printf("Error deleting vtep device: %s", err)
		}

		tenantName := tim.TenantName

		//Delete tenantStore and get tenantCIDR
		if err := ipam.DeleteTenantStore(tenantName); err != nil {
			log.Printf("Failed in deleting tenant store from node with err %s", err)
		}

		//Delete tenant from list in NodeStore and add to the avail list again
		nodeStore, err := ipam.NewNodeStore(defaultNodeDir, currentNodeName)
		if err != nil {
			log.Printf("Failed retrieving node store: %s", err)
		}
		nodeStore.LoadNodeData()
		nim, err := ipam.NewNodeIPAM(nodeStore, currentNodeName)
		if err != nil {
			log.Printf("Failed creating node IPAM: %s", err)
		}
		tenantList := nim.NodeStore.Data.TenantList
		availList := nim.NodeStore.Data.AvailableList

		for tenant, network := range tenantList {
			if tenant == tenantName {

				//Remove tenantCIDR and return to AvailList
				netS := network.String()
				availList = append(availList, netS)
				delete(tenantList, tenant)
			}

		}
		nim.NodeStore.Data.TenantList = tenantList
		nim.NodeStore.Data.AvailableList = availList
		nim.NodeStore.StoreNodeData()

		//Delete tenant from list in NodeStore and add to the avail list again
		return nil
	}

	return nil
}
