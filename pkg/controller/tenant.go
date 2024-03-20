package controller

import (
	"log"
	//"net"
	"time"

	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformer "github.com/jovik31/tenant/pkg/client/informers/externalversions/jovik31.dev/v1alpha1"
	tenantLister "github.com/jovik31/tenant/pkg/client/listers/jovik31.dev/v1alpha1"
	

	"github.com/jovik31/tenant/pkg/k8s"
	//bridge "github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/apimachinery/pkg/api/errors"
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

	defer utilruntime.HandleCrash()
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
	defer c.workqueue.Done(obj)

	objEvent, ok := obj.(*EventObject)
	if !ok {
		log.Printf("Failed in converting obj to EventObject: %s  in processing next item\n", obj)
		return false
	}

	if objEvent.eventType == "Add" {
		err:= c.addTenant(objEvent.key)
		if err != nil {
			log.Printf("Failed with error: %s  in adding tenant\n", err.Error())
			return false
		}

		c.workqueue.Forget(obj)
		return true
	}

	if objEvent.eventType == "Update" {

		err:= c.updateTenant(objEvent)
		if err != nil {
			log.Printf("Failed with error: %s  in updating tenant\n", err.Error())
			return false
		}
		c.workqueue.Forget(obj)
		return true

	}

	if objEvent.eventType == "Delete" {

		err:= c.deleteTenant(objEvent)
		if err != nil {
			log.Printf("Failed with error: %s  in deleting tenant\n", err.Error())
			return false
		}
		c.workqueue.Forget(obj)
		return true

	}else{


		log.Printf("Event is not of add, update or delete: Error %s  in processing next item\n", objEvent.eventType)
		c.workqueue.Forget(obj)
		return true
	}

}

func (c *Controller)addTenant(key string) error{

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
		return err
	}

	//Get the tenant by name
	tenant, err := c.tenantLister.Tenants(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Tenant not found: %s  in getting tenant by name\n", err.Error())
			return err
		}
		log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
		return err
	}
	//make changes on the copy of the tenant
	newTenant := tenant.DeepCopy()


	//Get clientset to get current node name
	kubeSet, err := k8s.GetKubeClientSet()
	if err != nil {
		log.Print("Error getting kube client set: ", err.Error())
	}

	currentNodeName, err := k8s.GetCurrentNodeName(kubeSet)
	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
	}

	//Check if the current node is part of the tenant
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
		nim.AllocateTenantCIDR(newTenant.Spec.Name)


		//To DO:
		//Add annotations to the node to show that the tenant is present on the node

		//Pods must have a enableTenant label and a tenant name annotation
		return nil
	} 

		return nil

}

		//log.Println("Node store availList: ", s.Data.AvailableList)
		//log.Println("Node store ip: ", s.Data.NodeIP)
		//nim, err := ipam.NewNodeIPAM(s, currentNodeName)
		//if err != nil {
			//log.Print("Error creating node IPAM: ", err.Error())
		//}
		//log.Println("Node IPAM created: ", nim.NodeName)
		//log.Println(nim.NodeStore.Data.AvailableList)

		//create the tenant file if the tenant is present on the node
		//res, err := bridge.CreateTenantBridge(newTenant.Spec.Name, 1500, &net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.IPv4Mask(255, 255, 255, 255)})
		//if err != nil {
			//log.Printf("Failed with error: %s  in creating tenant bridge\n", err.Error())
		//}
		//log.Print("Node exists in tenant", res)


func (c *Controller) updateTenant(obj *EventObject) error{
	log.Print(obj)

	return nil


}

func (c *Controller) deleteTenant(obj *EventObject) error{
	log.Print(obj)
	return nil

}