package controller




import (

	"log"
	"k8s.io/client-go/tools/cache"

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
