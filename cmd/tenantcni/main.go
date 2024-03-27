package main

import (

	"log"
	"github.com/containernetworking/cni/pkg/skel"
	//"github.com/containernetworking/cni/pkg/types"
	//current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	//"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	//"github.com/pkg/errors"


	//"github.com/jovik31/tenant/pkg/k8s"
)


const(

	plugin_name = "tenantcni"
)

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString(plugin_name))
}


func cmdAdd(args *skel.CmdArgs) error {


	log.Printf("cmdAdd details: containerID = %s, netNs = %s, ifName = %s, args = %s, path = %s, stdin = %s",
	args.ContainerID,
	args.Netns,
	args.IfName,
	args.Args,
	args.Path,
	string(args.StdinData),
)
	return nil

}

func cmdDel(args *skel.CmdArgs) error {
	
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	
	return nil
}