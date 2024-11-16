package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	tenantClientset "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	"github.com/jovik31/tenant/pkg/client/clientset/versioned/scheme"
	tenantInformer "github.com/jovik31/tenant/pkg/client/informers/externalversions/jovik31.dev/v1alpha1"
	tenantLister "github.com/jovik31/tenant/pkg/client/listers/jovik31.dev/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
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
		podSynced:    kubeInformer.Informer().HasSynced,
		tenantLister: tenantInformer.Lister(),
		podLister:    kubeInformer.Lister(),
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Tenant"),
		recorder:     recorder,
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

	logger := klog.FromContext(ctx)

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
