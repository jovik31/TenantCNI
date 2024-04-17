package backend

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"syscall"

	"github.com/vishvananda/netlink"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
)

func CreateTenantBridge(bridgeName string, mtu int, gateway netip.Addr) (netlink.Link, error) {
	if l, _ := netlink.LinkByName(bridgeName); l != nil {
		return l, nil
	}

	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   bridgeName,
			MTU:    mtu,
			TxQLen: -1,
		},
	}

	if err := netlink.LinkAdd(br); err != nil && err != syscall.EEXIST {
		return nil, err
	}

	dev, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return nil, err
	}
	gatewayString := gateway.String()
	gatewayString = gatewayString+"/24"

	ip, ipnet, err := net.ParseCIDR(gatewayString)
	if err != nil {
		return nil, err
	}
	if err := netlink.AddrAdd(dev, &netlink.Addr{IPNet: &net.IPNet{IP:ip, Mask: ipnet.Mask}}); err != nil {
		return nil, err
	}

	if err := netlink.LinkSetUp(dev); err != nil {
		return nil, err
	}

	return dev, nil
}

func DeleteTenantBridge(bridgeName string) error{
	bridge, err := netlink.LinkByName(bridgeName)
	if err!=nil{
		return err
	}
	return netlink.LinkDel(bridge)
}

func SetupVeth(netns ns.NetNS, br netlink.Link, mtu int, ifName string, podIP *net.IPNet, gateway net.IP) error {
	hostIface := &current.Interface{}
	err := netns.Do(func(hostNS ns.NetNS) error {
		
		// create the veth pair in the container and move host end into host netns
		hostVeth, containerVeth, err := ip.SetupVeth(ifName, mtu, "", hostNS)
		if err != nil {
			return err
		}
		hostIface.Name = hostVeth.Name

		// set ip for container veth
		conLink, err := netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(conLink, &netlink.Addr{IPNet: podIP}); err != nil {
			return err
		}

		// setup container veth
		if err := netlink.LinkSetUp(conLink); err != nil {
			return err
		}

		// add default route
		if err := ip.AddDefaultRoute(gateway, conLink); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", hostIface.Name, err)
	}

	if hostVeth == nil {
		return fmt.Errorf("nil hostveth")
	}

	// connect host veth end to the bridge
	if err := netlink.LinkSetMaster(hostVeth, br); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", hostVeth.Attrs().Name, br.Attrs().Name, err)
	}

	return nil
}

func DelVeth(netns ns.NetNS, ifName string) error {
	return netns.Do(func(ns.NetNS) error {
		l, err := netlink.LinkByName(ifName)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return netlink.LinkDel(l)
	})
}

func CheckVeth(netns ns.NetNS, ifName string, ip net.IP) error {
	return netns.Do(func(ns.NetNS) error {
		l, err := netlink.LinkByName(ifName)
		if err != nil {
			return err
		}

		ips, err := netlink.AddrList(l, netlink.FAMILY_V4)
		if err != nil {
			return err
		}

		for _, addr := range ips {
			if addr.IP.Equal(ip) {
				return nil
			}
		}

		return fmt.Errorf("failed to find ip %s for %s", ip, ifName)
	})
}