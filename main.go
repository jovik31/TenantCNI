package main

import (
	"fmt"
	"flag"
	"path/filepath"
	"log"
	"context"

	"github.com/jovik31/controller/pkg/utils"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/tools/clientcmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenant "github.com/jovik31/controller/pkg/client/clientset/versioned"
	
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


	utils.RegisterTenantCRD(config)
	tenantClient, err := tenant.NewForConfig(config)
	if err != nil {
		log.Printf("Error building tenant clientset: %s", err.Error())
	}
	
	tenants, err := tenantClient.Jovik31V1alpha1().Tenants("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting tenants: %s", err.Error())
	}
	fmt.Println(tenants)

}