Initial version of a k8s operator that registers a CRD to controll multi tenants within a cluster.


To DO:

    On delete event:
        Remove tenants in k8s
        Remove tenants in nodes:
            Remove network devices
            Remove network routes

Notes:
- Before deploying the tenantCNI make sure the node CIDRs defined by K8s are of a/22 mask. This is to allow each tenant the full 254 pods possible.
- Before deploying a pod to a custom tenant, make sure you have the customm tenant deployed. If not the pod will remain in a pending state.
      
TenantCNI deployment:
kubectl apply -f https://raw.githubusercontent.com/jovik31/TenantCNI/main/manifests/operator_deploy.yaml


It creates a default tenant, with a default bridge where containers are attached to.

To deploy a custom tenant check the file tenant_example.yaml file for tenant deployment. Then apply the tenant with the command:
kubectl apply -f https://raw.githubusercontent.com/jovik31/TenantCNI/tenant_example.yaml

When deploying pods to custom tenants it is mandatory to add a node selector and annotation to said pods. Check tenant_pod_example.yaml to see the annotation and node selector needed.
