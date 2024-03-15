package controller

import (
	"log"
	"time"

	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformer "github.com/jovik31/tenant/pkg/client/informers/externalversions/jovik31.dev/v1alpha1"
	tenantLister "github.com/jovik31/tenant/pkg/client/listers/jovik31.dev/v1alpha1"

	
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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
	c:= &Controller{
		tenantClient: 	tenantClient,
		tenantSynced: 	tenantInformer.Informer().HasSynced,
		tenantLister: 	tenantInformer.Lister(),
		workqueue: 		workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tenant"),
	}
	tenantInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.handleAdd,
			UpdateFunc: c.handleUpdate,
			DeleteFunc: c.handleDelete,
		},

	)
	//Add node informer for checking what tenants are available on a node at a specific time
	//Add the node informer to the controller and the Add and Delete functions for the cache ResourceEventHandlerFuncs
	//nodeInformer.Informer().AddEventHandler(

	return c
}


func (c *Controller) Run(ch chan struct{}) error{

	log.Print("Starting Tenant controller")
	if ok := cache.WaitForCacheSync(ch, c.tenantSynced); !ok{
		log.Println("Cache not synced")
	}
	go wait.Until(c.worker, time.Second, ch)
	<-ch
	return nil
}

func (c *Controller) worker(){ 
	for c.processNextItem(){


	}
}

func (c* Controller) processNextItem() bool{
	item, shut := c.workqueue.Get()
	if shut{
		//logs shut down
		return false
	}

	defer c.workqueue.Forget(item)

	key, err := cache.MetaNamespaceKeyFunc(item)
	if err != nil{
		log.Printf("Failed with error: %s  in calling key func on cached item", err.Error())
		return false
	}
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil{
		log.Printf("Failed with error: %s  in calling split meta namespace key", err.Error())
		return false
	}
	tenant, err := c.tenantLister.Tenants("default").Get(name)
	if err != nil{ 
		log.Printf("Failed with error: %s  in getting tenant by name", err.Error())
		return false
	}

	log.Printf("Processing tenant spec %v\n", tenant.Spec)

	return true
}

func (c *Controller) handleAdd(obj interface{}) {
	log.Print("Adding new tenant to the cluster")
	c.workqueue.Add(obj)
}

func (c *Controller) handleUpdate(oldObj, newObj interface{}) {
	log.Print("Updating a tenant on the cluster")
	c.workqueue.Add(newObj)
}

func (c *Controller) handleDelete(obj interface{}) {
	log.Print("Deleting a tenant from the cluster")
	c.workqueue.Add(obj)
}

