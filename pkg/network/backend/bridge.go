package backend

import (
	"net"
	"net/netip"
	"syscall"

	"github.com/vishvananda/netlink"
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

	_, ipnet, err := net.ParseCIDR(gatewayString)
	if err != nil {
		return nil, err
	}

	if err := netlink.AddrAdd(dev, &netlink.Addr{IPNet: ipnet}); err != nil {
		return nil, err
	}

	if err := netlink.LinkSetUp(dev); err != nil {
		return nil, err
	}

	return dev, nil
}
