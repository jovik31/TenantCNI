package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"regexp"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"

	llog "github.com/jovik31/tenant/pkg/log"
	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/jovik31/tenant/pkg/network/ipam"

	retry "github.com/jdvr/go-again"
)

const (
	plugin_name    = "tenantcni"
	defaultNodeDir = "/var/lib/cni/tenantcni"
)

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(plugin_name))

}

func cmdAdd(args *skel.CmdArgs) error {

	//Context for retry tenant loading
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	file := llog.LoadLogFile()
	defer file.Close()
	log.Print("Command: ADD")
	pod_name := get_regex(args.Args)

	//Fetches tenant name from the pod name, retries if failed
	tenant, err := retry.Retry[string](ctx, func(ctx context.Context) (string, error) {

		tenantName, err := getTenantPod(pod_name)
		if err != nil {
			log.Printf("Error getting tenant name: %s", err.Error())
			return "", err
		}
		if tenantName == "" {
			log.Printf("Tenant name not found, retrying")
			return "", errors.New("tenant name not found")
		}

		return tenantName, nil
	})

	if err != nil {
		log.Printf("Error getting tenant name: %s", err.Error())
		return err
	}
	//Debug
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
	ip, err := tim.AllocateIP(args.ContainerID, args.IfName, args.Netns, pod_name)
	if err != nil {
		log.Printf("Error allocating IP address: %s", err.Error())
		return err
	}
	log.Printf("Allocated IP: %s", ip.String())
	//Check if bridge exists, if not create:
	mtu := 1500
	br, err := backend.CreateTenantBridge(bridge, mtu, gateway)
	if err != nil {
		log.Print("Error creating bridge", err.Error())
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

	//Context for retry tenant loading
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	file := llog.LoadLogFile()
	defer file.Close()
	log.Print("Command: DEL")
	pod_name := get_regex(args.Args)

	//Fetches tenant name from the pod name, retries if failed
	tenant, err := retry.Retry[string](ctx, func(ctx context.Context) (string, error) {

		tenantName, err := getTenantPod(pod_name)
		if err != nil {
			log.Printf("Error getting tenant name: %s", err.Error())
			return "", err
		}
		if tenantName == "" {
			log.Printf("Tenant name not found, retrying")
			return "", errors.New("tenant name not found")
		}

		return tenantName, nil
	})
	if err != nil {
		log.Printf("Error getting tenant name: %s", err.Error())
		return err
	}
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
	if err := tim.ReleaseIP(args.ContainerID); err != nil {
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {

		log.Printf("Error getting namespace: %s", err.Error())
		return err
	}
	defer netns.Close()

	if err := deletePod(pod_name); err != nil {
		log.Printf("Error deleting pod: %s", err.Error())
		return err
	}

	return backend.DelVeth(netns, args.IfName)

}

func cmdCheck(args *skel.CmdArgs) error {

	//Context for retry tenant loading
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	file := llog.LoadLogFile()
	defer file.Close()
	log.Print("Command: CHECK")
	pod_name := get_regex(args.Args)

	//Fetches tenant name from the pod name, retries if failed
	tenant, err := retry.Retry[string](ctx, func(ctx context.Context) (string, error) {

		tenantName, err := getTenantPod(pod_name)
		if err != nil {
			log.Printf("Error getting tenant name: %s", err.Error())
			return "", err
		}
		if tenantName == "" {
			log.Printf("Tenant name not found, retrying")
			return "", errors.New("tenant name not found")
		}

		return tenantName, nil
	})
	if err != nil {
		log.Printf("Error getting tenant name: %s", err.Error())
		return err
	}
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
	ip, err := tim.CheckIP(args.ContainerID)
	if err != nil {
		log.Printf("Error checking IP: %s", err.Error())
		return err
	}
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		log.Printf("Error getting namespace: %s", err.Error())
		return err
	}
	defer netns.Close()

	return backend.CheckVeth(netns, args.IfName, ip)
}

// Check errors with regex
func get_regex(arg string) string {

	var re = regexp.MustCompile(`(-?)K8S_POD_NAME=(.+?)(;|$)`)
	mf := re.FindStringSubmatch(arg)
	return mf[2]

}

func getTenantPod(podname string) (string, error) {

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
		if name == podname && tenant != "" {
			return tenant, nil
		}
	}
	return "", nil
}

func deletePod(podname string) error {

	podStore, err := ipam.NewPodStore()
	if err != nil {
		log.Printf("Error creating pod store: %s", err.Error())
		return err
	}

	podStore.LoadPodData()
	pim, err := ipam.NewPodIPAM(podStore)
	if err != nil {
		log.Printf("Error creating pod ipam: %s", err.Error())
		return err
	}
	podData := pim.PodStore.Data
	podList := podData.Pods

	for name := range podList {
		if name == podname {
			delete(podList, name)
		}
	}
	podData.Pods = podList
	pim.PodStore.Data = podData
	pim.PodStore.StorePodData()
	return nil
}
