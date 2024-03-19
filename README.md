Initial version of a k8s operator that registers a CRD to controll multi tenants within a cluster.


To DO:
  ADD VNI field to tenant CUstom resource definition and custom resource

  On the default tenant custom resource add the dynamic finding of nodes so it deployes in every cluster node
  Controller:
    Check the possibility to use more thann 1 worker in the controller.
      Possible do to file mutex usage for node and tenant files
    Differentiate between update and add handlers
      On add event:
        Create tenantStore:
              - Add tenantCIDR
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
      
