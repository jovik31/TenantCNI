package main

import (
	//"context"
	"flag"
	//"fmt"
	"log"
	"path/filepath"
	"time"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	tenant "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformerFactory "github.com/jovik31/tenant/pkg/client/informers/externalversions"
	tenantController "github.com/jovik31/tenant/pkg/controller"
)


func main() {	

	var kubeconfig *string

	if home:=homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Printf("Error building kubeconfig: %s", err.Error())
	}


	tenantController.RegisterTenantCRD(config)
	tenantClient, err := tenant.NewForConfig(config)
	if err != nil {
		log.Printf("Error building tenant clientset: %s", err.Error())
	}
	
	//tenants, err := tenantClient.Jovik31V1alpha1().Tenants("").List(context.TODO(), metav1.ListOptions{})
	//if err != nil {
	//	log.Printf("Error getting tenants: %s", err.Error())
	//}
	//fmt.Println(tenants)


	//Start controller on a go routine
	ch:= make(chan struct{})
	informersFactory := tenantInformerFactory.NewSharedInformerFactory(tenantClient, 10*time.Minute)
	c := tenantController.NewController(tenantClient, informersFactory.Jovik31().V1alpha1().Tenants())
	informersFactory.Start(ch)
	if err := c.Run(ch); err != nil {
		log.Printf("Error running controller: %s\n", err.Error())
	}
	<-ch
}