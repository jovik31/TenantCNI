package controller

import (
	"log"
	"net"
	"reflect"

	"github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	"github.com/jovik31/tenant/pkg/k8s"
	"github.com/jovik31/tenant/pkg/network/ipam"
	"github.com/jovik31/tenant/pkg/network/routing"
	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

func (c *Controller) updateTenant(obj *EventObject) error {

	namespace, name, err := cache.SplitMetaNamespaceKey(obj.key)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
	}

	newTenant := obj.newObj.(*v1alpha1.Tenant)
	oldTenant := obj.oldObj.(*v1alpha1.Tenant)

	//Every change field is done on a copy of the real object
	kubeSet, err := k8s.GetKubeClientSet()
	if err != nil {
		log.Print("Error getting kube client set: ", err.Error())
	}

	currentNodeName, err := k8s.GetCurrentNodeName(kubeSet)
	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
	}

	//Check if it was a resync update,
	if reflect.DeepEqual(newTenant, oldTenant) {

		log.Printf("Resync update, no changes made to tenant: %s\n", newTenant.Name)

		//TO DO: Add logic for resync checks on the tenant routes and tenant interfaces
		return nil
	}

	//Check now for allowed changes. These are the only changes that can be made to a tenant
	if !reflect.DeepEqual(newTenant.Spec.Nodes, oldTenant.Spec.Nodes) {

		//If it did not exist in the past and it exists now, add the tenant to the node
		//Send add command to this nodes workerqueue
		if existsNode(newTenant.Spec.Nodes, currentNodeName) && !existsNode(oldTenant.Spec.Nodes, currentNodeName) {

			//Node was added to a tenant.
			//Create a new tenant in this object by sending a Add object to the workqueue.
			addObj := &EventObject{
				eventType: "Add",
				newObj:    obj.newObj,
				oldObj:    nil,
				key:       obj.key,
			}
			c.workqueue.Add(addObj)
			return nil
		}
		//If it the existed in the past and it does not exist now, delete the tenant from the node
		//Send delete command to this nodes workerqueue
		if !existsNode(newTenant.Spec.Nodes, currentNodeName) && existsNode(oldTenant.Spec.Nodes, currentNodeName) {

			//Node was added to a tenant.
			//Create a new tenant in this object by sending a Add object to the workqueue.
			delObj := &EventObject{
				eventType: "Delete",
				newObj:    nil,
				oldObj:    obj.oldObj,
				key:       obj.key,
			}
			c.workqueue.Add(delObj)
			return nil
		}

		//Check if the current node is part of the updated tenant
		if existsNode(newTenant.Spec.Nodes, currentNodeName) {

			//get the newest tenant version
			tenant, err := c.refreshTenant(namespace, name)
			if err != nil {
				log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
			}
			newTenant = tenant.DeepCopy()

			for _, node := range newTenant.Spec.Nodes {

				//Add arp, fdb and route entries for the remote vtep nodes, excludes current node

				if node.Name != currentNodeName {

					if node.NodeIP == "" || node.VtepIp == "" || node.VtepMac == "" {
						log.Printf("Node %s not initialized, update when node has been initialized", node.Name)
						continue
					}
					t, err := ipam.NewTenantStore(defaultNodeDir, newTenant.Name)
					if err != nil {
						log.Println("Error creating tenant store", err.Error())
					}
					t.LoadTenantData()
					tim, err := ipam.NewTenantIPAM(t, newTenant.Name)
					if err != nil {
						log.Println("Error creating tenant IPAM", err.Error())
					}
					//local vtep device information
					vtepName := tim.TenantStore.Data.Vxlan.VtepName
					vtepDevice, err := netlink.LinkByName(vtepName)
					if err != nil {
						log.Println("Error getting vtep device", err.Error())
					}
					vtepIndex := vtepDevice.Attrs().Index

					//remote vtep information
					vtepMac, err := net.ParseMAC(node.VtepMac)
					vtepIP := net.ParseIP(node.VtepIp)
					nodeIP := net.ParseIP(node.NodeIP)
					if err != nil {
						log.Println("Error parsing mac address", err.Error())
					}

					//Add arp, fdb and route entries for the remote vtep nodes
					err = routing.AddARP(vtepIndex, vtepIP, vtepMac)
					if err != nil {
						log.Println("Error adding arp entry", err.Error())
					}
					routing.AddFDB(vtepIndex, nodeIP, vtepMac)
					if err != nil {
						log.Println("Error adding fdb entry", err.Error())
					}
					mask := net.CIDRMask(newTenant.Spec.Prefix, 32)

					remoteCIDR := net.IPNet{IP: vtepIP, Mask: mask}
					err = routing.AddRoutes(vtepIndex, &remoteCIDR, vtepIP)
					if err != nil {
						log.Println("Error adding route entry", err.Error())
					}

					//Update tenant
				}
				continue

			}
			c.recorder.Event(newTenant, corev1.EventTypeNormal, "Update", "Tenant has been updated on node: "+currentNodeName)
		}

	}

	if existsNode(newTenant.Spec.Nodes, currentNodeName) {
		//Changing the Name, VNI or Prefix is not allowed. Revert changes with the ones applied at Tenant Addition.
		if !reflect.DeepEqual(newTenant.Spec.Prefix, oldTenant.Spec.Prefix) ||
			!reflect.DeepEqual(newTenant.Spec.VNI, oldTenant.Spec.VNI) ||
			!reflect.DeepEqual(newTenant.ObjectMeta.Name, oldTenant.ObjectMeta.Name) ||
			!reflect.DeepEqual(newTenant.Spec.Name, oldTenant.Spec.Name) {
			c.recorder.Event(newTenant, corev1.EventTypeWarning, "Failed Update", "Fields: Name, VNI and Prefix cannot be changed"+currentNodeName)

			//We need the values saved on node
			//Get tenant values saved on Node
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
			tenantOnFile := tim.TenantStore.Data

			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				tenant, err := c.refreshTenant(namespace, name)
				if err != nil {
					log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
					return err
				}

				//Updates resource with the correct values before illegal change in values of VNI, Prefix and Name
				err = c.UpdateTenantResource(tenant, tenantOnFile, namespace)
				return err
			})
			if err != nil {
				return err
			}
		}
	}

	return nil

}
