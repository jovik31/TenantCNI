package main

import (
	//"encoding/json"
	"net"
	"regexp"
	"os"
	"log"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	//"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	//"github.com/pkg/errors"
	"github.com/jovik31/tenant/pkg/network/ipam"
)


const(

	plugin_name = "tenantcni"
	logFile = "/var/log/tenantcni.log"
	//defaultPodFile= "/var/lib/cni/tenantcni/podlist/podlist.json"
)

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(plugin_name))
	//log.InitLogger(logFile)
	//log.Debugf("tenantcni plugin started")
}


func cmdAdd(args *skel.CmdArgs) error {

	file, err := openLogFile(logFile)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	log.Printf("cmdAdd args: %v\n", args.Args)
	
	pod_name := get_regex(args.Args)
	log.Printf("Pod name: %s", pod_name)

	podStore, err :=ipam.NewPodStore()
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
	podData:=pim.PodStore.Data
	podList:= podData.Pods
	log.Printf("Pod list: %v", podList)


	result := &current.Result{
		CNIVersion:current.ImplementedSpecVersion,
		IPs: []*current.IPConfig{
			{
				Address: net.IPNet{IP: net.ParseIP("10.10.10.2"),  Mask: net.CIDRMask(24, 32)},
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

func openLogFile(path string) (*os.File, error) {
    logFile, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
    if err != nil {
        return nil, err
    }
    return logFile, nil
}


func get_regex(arg string) string {

	var re = regexp.MustCompile(`(-?)K8S_POD_NAME=(.+?);`)
	mf:= re.FindStringSubmatch(arg)
	return mf[2]

}