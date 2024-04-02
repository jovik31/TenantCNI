package routing

import (
	"github.com/pkg/errors"
	"os/exec"
	"github.com/coreos/go-iptables/iptables"

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

func AddIptablesBridge(bridgeName, hostDeviceName, nodeCIDR string) error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}

	if err := ipt.AppendUnique("filter", "FORWARD", "-i", bridgeName, "-j", "ACCEPT"); err != nil {
		return err
	}
	return nil
}

func AddIptablesHost(hostDeviceName, nodeCIDR string, clusterCIDR string) error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	if err := ipt.AppendUnique("filter", "FORWARD", "-i", hostDeviceName, "-j", "ACCEPT"); err != nil {
		return err
	}
	if err := ipt.AppendUnique("nat", "POSTROUTING", "-s", clusterCIDR, "-d", clusterCIDR,"-j", "MASQUERADE"); err != nil {
		return err
	}

	return nil
}
//Check if this "iptables -P FORWARD ACCEPT" is needed

func IsolateTenant(tenant1CIDR string, tenant2CIDR string) error {
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	//Check if FORWARD is the correct chain to add this rule
	if err := ipt.AppendUnique("filter", "FORWARD", "-s", tenant1CIDR,"-d", tenant2CIDR, "-j", "DROP"); err != nil {
		return err
	}
	return nil
}
