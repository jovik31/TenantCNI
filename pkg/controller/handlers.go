package controller




import (

	"log"
	"k8s.io/client-go/tools/cache"
	v1 "k8s.io/api/core/v1"

)

func (c *Controller) handleAdd(obj interface{}) {

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("Failed in add tenant Handler: %s  in calling key func on cached item", err.Error())
	}
	
	addObj := &EventObject{
		eventType: 	"Add",
		newObj:    	obj,
		oldObj:   	nil,
		key:        key,
	}

	c.workqueue.Add(addObj)

}

func (c *Controller) handleUpdate(oldObj, newObj interface{}) {
	
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Printf("Failed in update tenant Handler: %s  in calling key func on cached item", err.Error())
	}
	updateObj := &EventObject{
		eventType: 	"Update",
		newObj:   	newObj,
		oldObj:   	oldObj,
		key:      	key,
	}

	c.workqueue.Add(updateObj)

}

func (c *Controller) handleDelete(obj interface{}) {

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("Deletion failed: %s  in calling key func on cached item", err.Error())
	}

	deleteObj := &EventObject{
		eventType: 	"Delete",
		newObj:   	obj,
		oldObj:   	nil,
		key:      	key,
	}
	
	c.workqueue.Add(deleteObj)

}


func (c *Controller) handlePodAdd(obj interface{}) {

	newPod := obj.(*v1.Pod)
	log.Printf("Pod Added: %s, with namespace %s", newPod.Name, newPod.Namespace)



}



func (c *Controller) handlePodUpdate(newobj interface{}, oldObj interface{}) {


}


func (c *Controller) handlePodDelete(obj interface{}) {

	oldObjPod := obj.(*v1.Pod)
	log.Printf("Pod Deleted: %s with namespace: %s",oldObjPod.Name, oldObjPod.Namespace)


}

