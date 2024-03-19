package controller

import (
	"log"
	"net"
	"time"

	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformer "github.com/jovik31/tenant/pkg/client/informers/externalversions/jovik31.dev/v1alpha1"
	tenantLister "github.com/jovik31/tenant/pkg/client/listers/jovik31.dev/v1alpha1"

	bridge "github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"
	"github.com/jovik31/tenant/pkg/k8s"

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

func (c *Controller) Run(ch chan struct{}) error {

	log.Print("Starting Tenant controller")
	if ok := cache.WaitForCacheSync(ch, c.tenantSynced); !ok {
		log.Println("Cache not synced")
	}
	go wait.Until(c.worker, time.Second, ch)
	<-ch
	return nil
}

func (c *Controller) worker() {
	for c.processNextItem() {

	}
}

// Processes the items that arrive on the workqueue
func (c *Controller) processNextItem() bool {

	key, shutdown := c.workqueue.Get()
	keyString := key.(string)
	if shutdown {
		//logs shut down
		return false
	}

	defer c.workqueue.Done(key)

	namespace, name, err := cache.SplitMetaNamespaceKey(keyString)
	if err != nil {
		log.Printf("Failed with error: %s  in splitting name and namespace from workqueue key", err.Error())
		return false
	}

	tenant, err := c.tenantLister.Tenants(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Tenant not found: %s  in getting tenant by name\n", err.Error())
			//c.workqueue.Forget(key)
			return true
		}
		log.Printf("Failed with error: %s  in getting tenant by name\n", err.Error())
		return false

	}

	newTenant := tenant.DeepCopy()
	kubeSet, err := k8s.GetKubeClientSet()
	if err != nil {
		log.Print("Error getting kube client set: ", err.Error())
	}
	currentNodeName, err := k8s.GetCurrentNodeName(kubeSet)

	if err != nil {
		log.Print("Error getting current node name: ", err.Error())
	}

	if existsNode(newTenant.Spec.Nodes, currentNodeName) {

		s, err :=ipam.NewNodeStore(defaultNodeDir,currentNodeName)
		if err != nil {
			log.Print("Error creating node store: ", err.Error())
		}
		s.LoadNodeData()
		log.Println("Node store availList: ", s.Data.AvailableList)
		log.Println("Node store ip: ", s.Data.NodeIP)
		nim, err := ipam.NewNodeIPAM(s, currentNodeName)
		if err != nil {
			log.Print("Error creating node IPAM: ", err.Error())
		}
		log.Println("Node IPAM created: ", nim.NodeName)
		log.Println(nim.NodeStore.Data.AvailableList)
		
		//create the tenant file if the tenant is present on the node
		res, err := bridge.CreateTenantBridge(newTenant.Spec.Name, 1500, &net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.IPv4Mask(255, 255, 255, 255)})
		if err != nil {
			log.Printf("Failed with error: %s  in creating tenant bridge\n", err.Error())
		}
		log.Print("Node exists in tenant", res)
	}

	c.workqueue.Forget(key)
	return true
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Printf("Adding a tenant to the cluster")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("Failed in add tenant Handler: %s  in calling key func on cached item", err.Error())
	}
	c.workqueue.Add(key)

}

func (c *Controller) handleUpdate(oldObj, newObj interface{}) {
	log.Print("Updating a tenant on the cluster")
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Printf("Failed in update tenant Handler: %s  in calling key func on cached item", err.Error())
	}
	c.workqueue.Add(key)

}

func (c *Controller) handleDelete(obj interface{}) {

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("Deletion failed: %s  in calling key func on cached item", err.Error())
	}
	log.Print("Deleting a tenant from the cluster", obj)
	c.workqueue.Add(key)

}
