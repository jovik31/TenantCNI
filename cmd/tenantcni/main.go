package main

import (
	//"encoding/json"
	"log"
	"net"
	"os"
	"regexp"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	//"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	//"github.com/pkg/errors"
	"github.com/jovik31/tenant/pkg/network/ipam"
)

const (
	plugin_name    = "tenantcni"
	logFile        = "/var/log/tenantcni.log"
	defaultNodeDir = "/var/lib/cni/tenantcni"

	//defaultPodFile= "/var/lib/cni/tenantcni/podlist/podlist.json"
)

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(plugin_name))
	//log.InitLogger(logFile)
	//log.Debugf("tenantcni plugin started")
}

func cmdAdd(args *skel.CmdArgs) error {

	file := loadLogFile()
	defer file.Close()

	pod_name := get_regex(args.Args)

	tenant, err := getTenantPod(pod_name)
	if err != nil {
		log.Printf("Error getting tenant name: %s", err.Error())
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
	//Allocate IP address from tenant
	//setup veth pairs and add them to the tenant bridge
	//Add default route to the namespace
	result := &current.Result{
		CNIVersion: current.ImplementedSpecVersion,
		IPs: []*current.IPConfig{
			{
				Address: net.IPNet{IP: net.ParseIP("10.10.10.2"), Mask: net.CIDRMask(24, 32)},
				Gateway: net.ParseIP("10.10.10.1"),
			},
		},
	}

	return types.PrintResult(result, result.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {

	return nil
}

func loadLogFile() *os.File {

	file, err := openLogFile(logFile)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	return file
}

func openLogFile(path string) (*os.File, error) {
	logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}

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
