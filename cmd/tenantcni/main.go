package main

import (
	//"encoding/json"
	"log"
	"net"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	//"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	//"github.com/pkg/errors"
	//"github.com/jovik31/tenant/pkg/log"
	//"github.com/jovik31/tenant/pkg/k8s"
)


const(

	plugin_name = "tenantcni"
	logFile = "/var/log/tenantcni.log"
)

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(plugin_name))
	//log.InitLogger(logFile)
	//log.Debugf("tenantcni plugin started")
}


func cmdAdd(args *skel.CmdArgs) error {

	log.Printf("cmdAdd args: %v\n", args.Args)
	
	log.Printf("cmdAdd details: containerID = %s, netNs = %s, ifName = %s, args = %s, path = %s, stdin = %s",
		args.ContainerID,
		args.Netns,
		args.IfName,
		args.Args,
		args.Path,
		string(args.StdinData),
	)
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