Initial version of a k8s operator that registers a CRD to controll multi tenants within a cluster.


To DO:

    Differentiate between update and add handlers
      On add event:
        Create tenantStore:
              - Add tenant Bridge IP (gateway)
              - Add tenant Bridge name
              Check if Vxlan is needed:
                -> Are there more than 1 node used to deploy the specific tenant ?
                   YES:
                     Deploy vxlan:
                         -> Add VNI used
                    -> Patch tenant CR with vxlan information
                  Patch node annotations with tenant enabled tag

      On update event:
      
