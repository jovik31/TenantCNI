package controller

import (
	"context"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"
	"github.com/vishvananda/netlink"

	"github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	"github.com/jovik31/tenant/pkg/client/clientset/versioned/scheme"
	tenantInformer "github.com/jovik31/tenant/pkg/client/informers/externalversions/jovik31.dev/v1alpha1"
	tenantLister "github.com/jovik31/tenant/pkg/client/listers/jovik31.dev/v1alpha1"
	"github.com/jovik31/tenant/pkg/k8s"
	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"
	"github.com/jovik31/tenant/pkg/network/routing"


	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	podInformers "k8s.io/client-go/informers/core/v1"
	podLister "k8s.io/client-go/listers/core/v1"
)

var (
	defaultNodeDir = "/var/lib/cni/tenantcni"
)

type Controller struct {
	//clientset for custom resource tenant
	tenantClient tenantClientset.Interface

	//kubeclientset for kubernetes API
	kubeClient kubernetes.Interface
	
	//tenant has synced
	tenantSynced cache.InformerSynced
	//pods has synced
	podSynced cache.InformerSynced

	//lister
	tenantLister tenantLister.TenantLister
	//pod lister
	podLister podLister.PodLister

	//queue
	workqueue workqueue.RateLimitingInterface

	//event recorder
	recorder record.EventRecorder
}

func NewController(
	ctx context.Context, 
	tenantClient tenantClientset.Interface, 
	kubeClient kubernetes.Interface,
	tenantInformer tenantInformer.TenantInformer, 
	kubeInformer podInformers.PodInformer) *Controller {

	logger := klog.FromContext(ctx)

	//Create event broadcaster -  add custom types to the default k8s scheme so events can be logged for tenant types
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")


	//Event broadcaster
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "tenant-controller"})

	c := &Controller{
		

		tenantClient: tenantClient,
		kubeClient:   kubeClient,
		tenantSynced: tenantInformer.Informer().HasSynced,
		podSynced: 	  kubeInformer.Informer().HasSynced,
		tenantLister: tenantInformer.Lister(),
		podLister:    kubeInformer.Lister(),
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tenant"),
		recorder: 	  recorder,
	}

	//Add tenant informer for checking what tenants are available on the cluster at a specific time
	logger.Info("Setting up tenant informer")
	tenantInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			UpdateFunc: c.handleUpdate,
			DeleteFunc: c.handleDelete,
		},
	)

	//Add pod informer for checking what pods are available on a node at a specific time
	logger.Info("Setting up pod informer")
	kubeInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handlePodAdd,
			UpdateFunc: c.handlePodUpdate,
			DeleteFunc: c.handlePodDelete,
		},
	)

	return c
}

func (c *Controller) Run(ctx context.Context) error {

	//Avoids panicking the controller
	defer utilruntime.HandleCrash()

	//makes sure the workqueue is shutdown, this triggers the workers to end
	defer c.workqueue.ShutDown()

	logger:=klog.FromContext(ctx)

	logger.Info("Starting tenant operator")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.tenantSynced, c.podSynced); !ok {
		log.Println("Cache not synced")
	}
	
	go wait.UntilWithContext(ctx, c.worker, time.Second)
	
	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")
	return nil
}

func (c *Controller) worker(ctx context.Context) {
	for c.processNextItem(ctx) {

	}
}

// Processes the items that arrive on the workqueue
func (c *Controller) processNextItem(ctx context.Context) bool {

	obj, shutdown := c.workqueue.Get()
	logger := klog.FromContext(ctx)
	if shutdown {
		log.Print("Worker queue is shudown")
		return false
	}

	//Indicate the queue we finished a task
	defer c.workqueue.Done(obj)

	objEvent, ok := obj.(*EventObject)
	if !ok {
		log.Printf("Failed in converting obj to EventObject: %s  in processing next item\n", obj)
		return false
	}

	var err error
	if objEvent.eventType == "Add" {
		err = c.addTenant(objEvent.key)
		if err == nil {
			//No errors in adding a tenant, tell the queue to stop tracking history for this object
			c.workqueue.Forget(obj)
			return true
		}

	}

	if objEvent.eventType == "Update" {

		err = c.updateTenant(objEvent)
		if err == nil {

			//No errors in updating a tenant, tell the queue to stop tracking history for this object
			c.workqueue.Forget(obj)
			return true
		}
	}

	if objEvent.eventType == "Delete" {

		err = c.deleteTenant(objEvent)
		if err == nil {
			//No errors in deleting a tenant, tell the queue to stop tracking history for this object
			c.workqueue.Forget(obj)
			return true
		}
		c.workqueue.Forget(obj)
		return true

	}
	if objEvent.eventType != "Add" && objEvent.eventType != "Update" && objEvent.eventType != "Delete" {

		log.Printf("Event is not of add, update or delete: Error %s  in processing next item\n", objEvent.eventType)
		c.workqueue.Forget(obj)
		return false
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", obj, err))

	c.workqueue.Forget(obj)
	logger.Info("Successfully synced")

	return true

}

func (c *Controller) addTenant(key string) error {

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
		return err
	}

	//Get the newest tenant version
	tenant, err := c.refreshTenant(namespace, name)
	if err != nil {
		log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
		return err
	}
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
		if (len(newTenant.Spec.Nodes) > 1){
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
			node, err:= k8s.GetCurrentNode(kubeSet, currentNodeName)
			if err!= nil{
				log.Printf("Failed to retrive node")
			}
	
			err= k8s.StoreTenantAnnotationNode(kubeSet, node, tenant.Name)
			return err
		})
		if err != nil {
			log.Println("Failed to update Node resource on API")
			return err
		}

		//Allow tenant traffic forwarding
		if err := routing.AllowForwardingTenant(tim.TenantStore.Data.TenantCIDR); err!=nil{
			log.Println("Failed to allow tenant traffic forwarding")
		}

		//Block traffic between tenants on the same node
		//1st: Get all tenants present in the node using the nodeStore
		tenantList := nim.NodeStore.Data.TenantList

		for tenant, tenantCIDR := range tenantList {
			if tenant != tim.TenantName && tenant != "defaulttenant" {
				stenantCIDR :=tenantCIDR.String()
				routing.BlockTenant2TenantTraffic(tim.TenantStore.Data.TenantCIDR, stenantCIDR)
				log.Printf("Blocked traffic between tenants: %s and %s", tim.TenantName, tenant)

			}else{
				continue
			}
		}
		c.recorder.Event(newTenant, corev1.EventTypeNormal, "Add", "Tenant has been created on node: " + currentNodeName)
		return nil
	}


	return nil

}
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
	if(reflect.DeepEqual(newTenant, oldTenant)){

		log.Printf("Resync update, no changes made to tenant: %s\n", newTenant.Name)

		//TO DO: Add logic for resync checks on the tenant routes and tenant interfaces
		return nil
	}

	//Check now for allowed changes. These are the only changes that can be made to a tenant
	if (!reflect.DeepEqual(newTenant.Spec.Nodes, oldTenant.Spec.Nodes)){

		//If it the existed in the past and it does not exist now, delete the tenant from the node
		//Send delete command to this nodes workerqueue

		//If it did not exist in the past and it exists now, add the tenant to the node
		//Send add command to this nodes workerqueue

		//Check if the current node is part of the updated tenant
		if(existsNode(newTenant.Spec.Nodes, currentNodeName)){


			//get the newest tenant version
			tenant, err := c.refreshTenant(namespace, name)
			if err != nil {
				log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
			}
			newTenant = tenant.DeepCopy()

			for _, node := range(newTenant.Spec.Nodes){

				//Add arp, fdb and route entries for the remote vtep nodes, excludes current node

				if(node.Name != currentNodeName){

					if node.NodeIP == "" || node.VtepIp == "" || node.VtepMac == ""{
						log.Printf("Node %s not initialized, update when node has been initialized", node.Name)
						continue
					}
					t, err := ipam.NewTenantStore(defaultNodeDir, newTenant.Name)
					if err != nil {
						log.Println("Error creating tenant store", err.Error())
					}
					t.LoadTenantData()
					tim, err := ipam.NewTenantIPAM(t, newTenant.Name)
					if err!=nil{
						log.Println("Error creating tenant IPAM", err.Error())
					}
					//local vtep device information
					vtepName := tim.TenantStore.Data.Vxlan.VtepName
					vtepDevice, err:=netlink.LinkByName(vtepName)
					if err!=nil{
						log.Println("Error getting vtep device", err.Error())
					}
					vtepIndex:=vtepDevice.Attrs().Index

					//remote vtep information
					vtepMac, err := net.ParseMAC(node.VtepMac)
					vtepIP := net.ParseIP(node.VtepIp)
					nodeIP := net.ParseIP(node.NodeIP)
					if err!=nil{
						log.Println("Error parsing mac address", err.Error())
					}

					//Add arp, fdb and route entries for the remote vtep nodes
					err=routing.AddARP(vtepIndex, vtepIP, vtepMac)
					if err!=nil{
						log.Println("Error adding arp entry", err.Error())
					}
					routing.AddFDB(vtepIndex, nodeIP, vtepMac)
					if err!=nil{
						log.Println("Error adding fdb entry", err.Error())
					}
					mask := net.CIDRMask(newTenant.Spec.Prefix, 32)
				
					remoteCIDR := net.IPNet{IP: vtepIP, Mask:mask}
					err =routing.AddRoutes(vtepIndex, &remoteCIDR, vtepIP)
					if err!=nil{
						log.Println("Error adding route entry", err.Error())
					}	

					//Update tenant
				}
				continue

			}
			c.recorder.Event(newTenant, corev1.EventTypeNormal, "Update", "Tenant has been updated on node: " + currentNodeName)
		}

	}

	if(existsNode(newTenant.Spec.Nodes, currentNodeName)){
	//Changing the Name, VNI or Prefix is not allowed. Revert changes with the ones applied at Tenant Addition.
		if(!reflect.DeepEqual(newTenant.Spec.Prefix, oldTenant.Spec.Prefix) || 
			!reflect.DeepEqual(newTenant.Spec.VNI, oldTenant.Spec.VNI)||
			!reflect.DeepEqual(newTenant.ObjectMeta.Name, oldTenant.ObjectMeta.Name) || 
			!reflect.DeepEqual(newTenant.Spec.Name, oldTenant.Spec.Name)){
			c.recorder.Event(newTenant, corev1.EventTypeWarning, "Failed Update", "Fields: Name, VNI and Prefix cannot be changed" + currentNodeName)

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

func (c *Controller) deleteTenant(obj *EventObject) error {

	//Logic for tenant deletion
	//Check if node is part of the tenant
	//If it is, delete tenant file, remove vxlam devices and remove node annotation
	deletedTenant := obj.oldObj.(*v1alpha1.Tenant)

	namespace, name, err := cache.SplitMetaNamespaceKey(obj.key)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
		return err
	}

	//Get the newest tenant version
	tenant, err := c.refreshTenant(namespace, name)
	if err!= nil{
		if errors.IsNotFound(err) {
			log.Printf("Tenant not found: %s, deleted from K8s API ", deletedTenant.Name)
		}else {
			log.Printf("Failed with error: %s,", err.Error())
			return err
		}
	}
	k8sClient, err := k8s.GetKubeClientSet()
	if err != nil {
		log.Print("Error getting kube client set: ", err.Error())
		return err
	}
	currentNodeName, err:= k8s.GetCurrentNodeName(k8sClient)
	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
		return err
	}

	if(existsNode(deletedTenant.Spec.Nodes, currentNodeName)){

		//Get tenant store
		//Delete tenant vxlan, delete tenant bridge and delete tenantfile
		
		return nil


	}
	log.Print(tenant)
	return nil

}
