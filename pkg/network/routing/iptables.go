package routing

import (
	"log"

	"os/exec"

	"github.com/coreos/go-iptables/iptables"
	"github.com/pkg/errors"
)

// Enables ip fowarding on the host
func EnableIPForwarding() error {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Failed to enable IP forwarding")
	}
	return nil
}

func AllowBridgeForward(bridgeInterface string) error {

	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		log.Printf("Error creating iptables: %s", err.Error())
		return err
	}
	//Add rule to allow forwarding from bridge to host
	if err := ipt.AppendUnique("filter", "FORWARD", "-i", bridgeInterface, "-j", "ACCEPT"); err != nil {
		log.Printf("Error adding iptables rule: %s", err.Error())
		return err
	}
	return nil
}

//func AllowPostRouting(nodeCIDR string) error{
//
//	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
//	if err != nil {
//		log.Printf("Error creating iptables: %s", err.Error())
//		return err
//	}
//Add rule to allow source nat to access external networks
//	if err := ipt.AppendUnique("nat", "POSTROUTING", "-s", nodeCIDR, "-j", "MASQUERADE"); err != nil {
//		log.Printf("Error adding iptables rule: %s", err.Error())
//		return err
//	}
//	return nil
//}

func AllowForwardingTenant(TenantCIDR string) error {

	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		log.Printf("Error creating iptables: %s", err.Error())
		return err
	}
	//Add rule to allow forwarding from tenant
	if err := ipt.AppendUnique("filter", "FORWARD", "-s", TenantCIDR, "-j", "ACCEPT"); err != nil {
		log.Printf("Error adding iptables rule: %s", err.Error())
		return err
	}
	//Add rule to allow forwarding to tenant
	if err := ipt.AppendUnique("filter", "FORWARD", "-d", TenantCIDR, "-j", "ACCEPT"); err != nil {
		log.Printf("Error adding iptables rule: %s", err.Error())
		return err
	}
	return nil
}

func BlockTenant2TenantTraffic(tenant1CIDR string, tenant2CIDR string) error {

	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		log.Printf("Error creating iptables: %s", err.Error())
		return err
	}
	//Add rule to block traffic between tenants
	if err := ipt.InsertUnique("filter", "FORWARD", 1, "-s", tenant1CIDR, "-d", tenant2CIDR, "-j", "DROP"); err != nil {
		log.Printf("Error adding iptables rule: %s", err.Error())
		return err
	}
	if err := ipt.InsertUnique("filter", "FORWARD", 1, "-s", tenant2CIDR, "-d", tenant1CIDR, "-j", "DROP"); err != nil {
		log.Printf("Error adding iptables rule: %s", err.Error())
		return err
	}

	return nil

}
