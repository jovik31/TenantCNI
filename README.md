Initial version of a k8s operator that registers a CRD to controll multi tenants within a cluster.


To DO:

    On delete event:
        Remove tenants in k8s
        Remove tenants in nodes:
            Remove network devices
            Remove network routes
        Remove pods 

    CNI Part:
        Bridging and IP management
        Isolate traffic between tenants within the same node. Except comms to the default tenant
    
      
