package main

import (
	"encoding/json"
	"log"
	"net"
	"regexp"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	
	llog "github.com/jovik31/tenant/pkg/log"
	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"
)

const (
	plugin_name    = "tenantcni"
	defaultNodeDir = "/var/lib/cni/tenantcni"

)

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(plugin_name))
	//log.InitLogger(logFile)
	//log.Debugf("tenantcni plugin started")
}

func cmdAdd(args *skel.CmdArgs) error {

	file := llog.LoadLogFile()
	defer file.Close()

	pod_name := get_regex(args.Args)

	tenant, err := getTenantPod(pod_name)
	if err != nil {
		log.Printf("Error getting tenant name: %s", err.Error())
		return err
	}
	if tenant == "" {
		log.Printf("Tenant name not found")
		return err
	}
	log.Printf("Pod name %s, Tenant name: %s", pod_name, tenant)

	//With tenant name get tenant store
	tenantStore, err := ipam.NewTenantStore(defaultNodeDir, tenant)
	if err != nil {
		log.Printf("Error creating tenant store: %s", err.Error())
		return err
	}
	tenantStore.LoadTenantData()
	tim, err := ipam.NewTenantIPAM(tenantStore, tenant)
	if err != nil {
		log.Printf("Error creating tenant ipam: %s", err.Error())
		return err
	}
	//get tenant bridge name and gateway
	gateway := tim.TenantStore.Data.Bridge.Gateway
	bridge := tim.TenantStore.Data.Bridge.Name
	log.Printf("Bridge: %s, Gateway: %s", bridge, gateway)


	//Allocate IP address from the specific tenant to the next Pod
	ip, err := tim.AllocateIP(args.ContainerID, args.IfName)
	if err != nil {
		log.Printf("Error allocating IP address: %s", err.Error())
		return err
	}
	log.Printf("Allocated IP: %s", ip.String())
	//Check if bridge exists, if not create:
	mtu := 1500
	br, err := backend.CreateTenantBridge(bridge, mtu, gateway)
	if err != nil {
		log.Printf("Error creating bridge: %s", err.Error())
		return err
	}
	log.Printf("Bridge created: %s", br.Attrs().Name)

	//Get namespace
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return err
	}
	defer netns.Close()

	gatewayString := gateway.String()
	gtw := net.ParseIP(gatewayString)

	if err := backend.SetupVeth(netns, br, mtu, args.IfName, tim.IPNet(ip), gtw); err != nil {
		log.Printf("Error setting up veth: %s", err.Error())
		return err
	}

	result := &current.Result{
		CNIVersion: "0.3.1",
		IPs: []*current.IPConfig{
			{
				Address: net.IPNet{IP: ip, Mask: net.CIDRMask(24, 32)},
				Gateway: net.ParseIP(gateway.String()),
			},
		},
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	log.Printf("Result: %s", resultBytes)
	return types.PrintResult(result, result.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {

	file := llog.LoadLogFile()
	defer file.Close()

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {

	file := llog.LoadLogFile()
	defer file.Close()
	return nil
}

//Check errors with regex
func get_regex(arg string) string {

	var re = regexp.MustCompile(`(-?)K8S_POD_NAME=(.+?);`)
	mf := re.FindStringSubmatch(arg)
	return mf[2]

}

func getTenantPod(podname string) (string, error) {

	var tenantName string
	podStore, err := ipam.NewPodStore()
	if err != nil {
		log.Printf("Error creating pod store: %s", err.Error())
		return "", err
	}

	podStore.LoadPodData()
	pim, err := ipam.NewPodIPAM(podStore)
	if err != nil {
		log.Printf("Error creating pod ipam: %s", err.Error())
		return "", err
	}
	podData := pim.PodStore.Data
	podList := podData.Pods

	for name, tenant := range podList {
		if name == podname {
			tenantName = tenant
		}
	}
	if tenantName == "" {
		return "", err
	}
	return tenantName, nil
}
