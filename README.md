Initial version of a k8s operator that registers a CRD to controll multi tenants within a cluster.


To DO:

  Make operator run in a pod:
    - Add in cluster config
    - Add RBAC policies
    - Create docker image
    - Create yaml deployments
Tenant controll:
    - Add Informers and watchers
    - Add, Update, Delete eventHandlers
    - Network access to node and capabilities
