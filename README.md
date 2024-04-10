Initial version of a k8s operator that registers a CRD to controll multi tenants within a cluster.


To DO:

    On delete event:
        Remove tenants in k8s
        Remove tenants in nodes:
            Remove network devices
            Remove network routes
        Remove pods 
    
      
TenantCNI deployment:

kubectl apply -f https://raw.githubusercontent.com/jovik31/TenantCNI/main/manifests/operator_deploy.yaml
