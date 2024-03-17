package main

import (
	"context"

	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenant "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformerFactory "github.com/jovik31/tenant/pkg/client/informers/externalversions"
	tenantController "github.com/jovik31/tenant/pkg/controller"
	tenantRegistration "github.com/jovik31/tenant/pkg/crd"
	kubecnf "github.com/jovik31/tenant/pkg/k8s"
)

var (
	defaultTenantDir = "/var/cni/tenants/"
)

func main() {


	//init kubernetes client for initial configurations
	config, err := kubecnf.InitKubeConfig()
	if err != nil {
		log.Printf("Error building kubernetes clientset: %s", err.Error())
	}

	//Register tenant CRD onto the kubernetes API
	tenantRegistration.RegisterTenantCRD(config)

	//Register default tenant in the k8s API
	tenantClient, err := tenant.NewForConfig(config)
	if err != nil {
		log.Printf("Error building tenant clientset: %s", err.Error())
	}
	tenantRegistration.RegisterDefaultTenant(tenantClient)

	//Get current node CIDR to populate the node management file

	nodes, err := kubecnf.GetKubeClientSet().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting nodes: %s", err.Error())
	}
	log.Print(nodes.Items[0].Spec.PodCIDR)


	//Start controller on a go routine
	ch := make(chan struct{})
	informersFactory := tenantInformerFactory.NewSharedInformerFactory(tenantClient, 10*time.Minute)
	c := tenantController.NewController(tenantClient, informersFactory.Jovik31().V1alpha1().Tenants())
	informersFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("Error running controller: %s\n", err.Error())
	}
	<-ch

}
