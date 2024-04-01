package routing

import (
	"github.com/pkg/errors"
	"os/exec"

	//"github.com/coreos/go-iptables/iptables"
)




//func AddIpTablesDocker() error {
//	return nil
//}


//func AddIpTablesTenants() error {

	//ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	//if err != nil {
	//	return errors.Wrapf(err,"Failed to create iptables")
	//}
	//err = ipt.Append("nat", "POSTROUTING", "-s", ")
	//return nil
//}

//Enables ip fowarding on the host
func EnableIPForwarding() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
	return errors.Wrapf(err,"Failed to enable IP forwarding")
}
	return nil
}