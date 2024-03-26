package controller

import (
	"context"
	"fmt"
	"log"
	"reflect"

	//"net"
	"time"

	"github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"
	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformer "github.com/jovik31/tenant/pkg/client/informers/externalversions/jovik31.dev/v1alpha1"
	tenantLister "github.com/jovik31/tenant/pkg/client/listers/jovik31.dev/v1alpha1"

	"github.com/jovik31/tenant/pkg/k8s"
	//bridge "github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

var (
	defaultNodeDir = "/var/lib/cni/tenantcni"
)

type Controller struct {
	//clientset for custom resource tenant
	tenantClient tenantClientset.Interface
	//tenant has synced
	tenantSynced cache.InformerSynced
	//lister
	tenantLister tenantLister.TenantLister
	//queue
	workqueue workqueue.RateLimitingInterface
}

func NewController(tenantClient tenantClientset.Interface, tenantInformer tenantInformer.TenantInformer) *Controller {
	c := &Controller{
		tenantClient: tenantClient,
		tenantSynced: tenantInformer.Informer().HasSynced,
		tenantLister: tenantInformer.Lister(),
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tenant"),
	}
	tenantInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			UpdateFunc: c.handleUpdate,
			DeleteFunc: c.handleDelete,
		},
	)
	//Add node informer for checking what tenants are available on a node at a specific time
	//Add the node informer to the controller and the Add and Delete functions for the cache ResourceEventHandlerFuncs
	//nodeInformer.Informer().AddEventHandler(

	return c
}

func (c *Controller) Run(ch chan struct{}, workers int) error {

	//Avoids panicking the controller
	defer utilruntime.HandleCrash()

	//makes sure the workqueue is shutdown, this triggers the workers to end
	defer c.workqueue.ShutDown()

	log.Print("Starting Tenant controller")
	if ok := cache.WaitForCacheSync(ch, c.tenantSynced); !ok {
		log.Println("Cache not synced")
	}

	log.Println("Starting workers", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, ch)
	}
	log.Println("Started workers")
	<-ch
	log.Println("Shutting down workers")
	return nil
}

func (c *Controller) worker() {
	for c.processNextItem() {

	}
}

// Processes the items that arrive on the workqueue
func (c *Controller) processNextItem() bool {

	obj, shutdown := c.workqueue.Get()
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
		nim.AllocateTenant(newTenant.Spec.Name, newTenant.Spec.VNI)

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
		return nil
	}

	return nil

}
func (c *Controller) updateTenant(obj *EventObject) error {

	newTenant := obj.newObj.(*v1alpha1.Tenant)
	oldTenant := obj.oldObj.(*v1alpha1.Tenant)
	//Check if it was a resync update
	if(newTenant == oldTenant){

		log.Printf("Resync update, tenant is the same")
		return nil
	}
	//Check if this node is allocated to the tenant


	//If it reaches here, a change was made
	if (!reflect.DeepEqual(newTenant.Spec.Nodes, oldTenant.Spec.Nodes)){

			log.Println("No updates to node list of tenant")

		return nil



	}

	





	//First thing, check if current node exists on the Tenant

	return nil

}

func (c *Controller) deleteTenant(obj *EventObject) error {
	log.Print(obj)
	return nil

}
