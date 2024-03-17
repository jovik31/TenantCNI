package routing

import (
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

func AddArp(localVtepID int, remoteVtepIP net.IP, remoteVtepMac net.HardwareAddr) error {

	return netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    localVtepID,
		State:        netlink.NUD_PERMANENT,
		Type:         syscall.RTN_UNICAST,
		IP:           remoteVtepIP,
		HardwareAddr: remoteVtepMac,
	})
}
