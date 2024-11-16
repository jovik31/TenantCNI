package main

import (
	"encoding/json"
	"log"
	"time"

	tenant "github.com/jovik31/tenant/pkg/client/clientset/versioned"
	tenantInformerFactory "github.com/jovik31/tenant/pkg/client/informers/externalversions"
	tenantController "github.com/jovik31/tenant/pkg/controller"
	tenantRegistration "github.com/jovik31/tenant/pkg/crd"
	kubecnf "github.com/jovik31/tenant/pkg/k8s"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/sample-controller/pkg/signals"

	confType "github.com/jovik31/tenant/pkg/apis/jovik31.dev/v1alpha1"

	"github.com/jovik31/tenant/pkg/network/ipam"
	"github.com/jovik31/tenant/pkg/network/routing"
)

var (
	defaultNodeDir = "/var/lib/cni/tenantcni"
	confMap        = "tenantcni-config"
)

func main() {

	log.Println("Starting tenant operator")
	ctx := signals.SetupSignalHandler()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	//init kubernetes client for initial configurations
	config, err := kubecnf.InitKubeConfig()
	if err != nil {
		log.Printf("Error building kubernetes clientset: %s", err.Error())
	}

	//Register tenant CRD onto the kubernetes API using the rest Configuration
	tenantRegistration.RegisterTenantCRD(config)

	kubeclientset, err := kubecnf.GetKubeClientSet()
	if err != nil {
		log.Printf("Error building kubernetes clientset: %s", err.Error())
	}

	//Get current node name
	currentNodeName, err := kubecnf.GetCurrentNodeName(kubeclientset)
	if err != nil {
		log.Printf("Error getting current node name: %s", err.Error())
	}
	//Get node list
	nodeList, err := kubecnf.GetNodes(kubeclientset)
	if err != nil {
		log.Printf("Error getting node list: %s", err.Error())
	}
	//Get current node CIDR
	nodeCIDR, err := kubecnf.GetNodeCIDR(nodeList, currentNodeName)
	if err != nil {
		log.Printf("Error getting node CIDR: %s", err.Error())
	}
	currentNodeIP, err := kubecnf.GetCurrentNodeIP(kubeclientset, currentNodeName)
	if err != nil {
		log.Printf("Error getting current node IP: %s", err.Error())
	}
	log.Printf("Current node IP: %s", currentNodeIP)

	//Create a new node store for the current node with the nodeCIDR
	nodeStore, err := ipam.NewNodeStore(defaultNodeDir, currentNodeName)
	if err != nil {
		log.Printf("Error creating node store: %s", err.Error())
	}

	nim, err := ipam.NewNodeIPAM(nodeStore, currentNodeName)
	if err != nil {
		log.Printf("Error creating node IPAM: %s", err.Error())
	}

	nim.NodeStore.AddNodeIP(currentNodeIP)
	nim.NodeStore.AddNodeCIDR(nodeCIDR)

	//List all available subnets for tenants in the current node
	availList := ipam.ListSubnets(nodeCIDR, 24)
	nim.NodeStore.AddAvailableTenantList(availList)

	//Get tenant client to start  controller and to be able to register our default tenant
	tenantClient, err := tenant.NewForConfig(config)
	if err != nil {
		log.Printf("Error building tenant clientset: %s", err.Error())
	}

	//Register default tenant in the k8s API
	tenantRegistration.RegisterDefaultTenant(tenantClient, nodeList)

	//Get info from config map on the cluster PodCIDR for iptables.
	netConf, err := kubecnf.GetConfigMap(kubeclientset, "default", confMap)
	if err != nil {
		log.Printf("Error retrieving config from map %s", err)
	}

	//Get the PodCIDR from the config map
	configMap := &confType.ConfMap{}
	confString := netConf.Data["net-conf.json"]
	err = json.Unmarshal([]byte(confString), &configMap)
	if err != nil {
		log.Printf("Error unmarshalling config map: %s", err.Error())
	}
	log.Printf("PodCIDR: %s", configMap.PodCIDR)

	//Enable post routing for the node CIDR
	//if err := routing.AllowPostRouting(nodeCIDR); err != nil {
	//	log.Printf("Error enabling post routing: %s", err.Error())
	//}

	//enable IPv4 forwarding, if not enabled
	if err := routing.EnableIPForwarding(); err != nil {
		log.Printf("Error enabling IP forwarding: %s", err.Error())
	}

	//Start controller on a go routine
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, 10*time.Second)
	tInformersFactory := tenantInformerFactory.NewSharedInformerFactory(tenantClient, 10*time.Minute)

	c := tenantController.NewController(ctx, tenantClient, kubeclientset,
		tInformersFactory.Jovik31().V1alpha1().Tenants(), kubeInformerFactory.Core().V1().Pods())

	tInformersFactory.Start(ctx.Done())
	kubeInformerFactory.Start(ctx.Done())

	if err := c.Run(ctx); err != nil {
		log.Printf("Error running controller: %s\n", err.Error())
	}

}
