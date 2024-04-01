package backend
import (
	"crypto/rand"
	"net"
	"strings"
	"syscall"
	"log"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

func NewHardwareAddr() (net.HardwareAddr, error) {
	hardwareAddr := make(net.HardwareAddr, 6)
	if _, err := rand.Read(hardwareAddr); err != nil {
		return nil, errors.Wrap(err, "read hardware addr error")
	}

	// ensure that address is locally administered and unicast
	hardwareAddr[0] = (hardwareAddr[0] & 0xfe) | 0x02

	return hardwareAddr, nil
}

func getIfaceAddr(iface *net.Interface) ([]netlink.Addr, error) {
	return netlink.AddrList(&netlink.Device{
		LinkAttrs: netlink.LinkAttrs{
			Index: iface.Index,
		},
	}, syscall.AF_INET)
}

func getDefaultGatewayInterface() (*net.Interface, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return nil, errors.Wrap(err, "RouteList error")
	}

	for _, route := range routes {
		if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
			if route.LinkIndex <= 0 {
				return nil, errors.Errorf("found default route but could not determine interface")
			}
			return net.InterfaceByIndex(route.LinkIndex)
		}
	}

	return nil, errors.Errorf("unable to find default route")
}

const (
	
	vxlanPort     = 8472
	encapOverhead = 50
)

func newVxlanDevice(vtepName string, vni int, vtepMac string) (*netlink.Vxlan, error) {
	hardwareAddr, err := net.ParseMAC(vtepMac)
	if err != nil {
		return nil, errors.Wrap(err, "ParseMAC error")
	}
	//hardwareAddr, err := NewHardwareAddr()
	if err != nil {
		return nil, errors.Wrap(err, "newHardwareAddr error")
	}

	gateway, err := getDefaultGatewayInterface()
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultGatewayInterface error")
	}

	localHostAddrs, err := getIfaceAddr(gateway)
	if err != nil {
		return nil, errors.Wrap(err, "getIfaceAddr error")
	}

	if len(localHostAddrs) == 0 {
		return nil, errors.Errorf("length of local host addrs is 0")
	}

	return ensureVxlan(&netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:         vtepName,
			HardwareAddr: hardwareAddr,
			MTU:          gateway.MTU - encapOverhead,
		},
		VxlanId:      vni,
		VtepDevIndex: gateway.Index,
		SrcAddr:      localHostAddrs[0].IP,
		Port:         vxlanPort,
	})
}

func ensureVxlan(vxlan *netlink.Vxlan) (*netlink.Vxlan, error) {
	link, err := netlink.LinkByName(vxlan.Name)
	if err == nil {
		v, ok := link.(*netlink.Vxlan)
		if !ok {
			return nil, errors.Errorf("link %s already exists but not vxlan device", vxlan.Name)
		}

		log.Printf("vxlan device %s already exists", vxlan.Name)
		return v, nil
	}

	if !strings.Contains(err.Error(), "Link not found") {
		return nil, errors.Wrapf(err, "get link %s error", vxlan.Name)
	}

		log.Printf("vxlan device %s not found, and create it", vxlan.Name)

	if err = netlink.LinkAdd(vxlan); err != nil {
		return nil, errors.Wrap(err, "LinkAdd error")
	}

	link, err = netlink.LinkByName(vxlan.Name)
	if err != nil {
		return nil, errors.Wrap(err, "LinkByName error")
	}

	return link.(*netlink.Vxlan), nil
}


func InitVxlanDevice(podCidr string, vtepName string, vni int, vtepMac string) (*netlink.Vxlan, error) {
	
	vxlanLink, err := newVxlanDevice(vtepName, vni, vtepMac)
	if err != nil {
		return nil, errors.Wrap(err, "NewVXLANDevice error")
	}

	_, cidr, err := net.ParseCIDR(podCidr)
	if err != nil {
		return nil, errors.Wrap(err, "ParseCIDR error")
	}

	existingAddrs, err := netlink.AddrList(vxlanLink, netlink.FAMILY_V4)
	if err != nil {
		return nil, errors.Wrapf(err, "AddrList error")
	}
	if len(existingAddrs) == 0 {
		
		log.Printf("config vxlan device %s ip: %s", vxlanLink.Name, cidr.IP)
		if err = netlink.AddrAdd(vxlanLink, &netlink.Addr{
			IPNet: &net.IPNet{
				IP:   cidr.IP,
				Mask: net.IPv4Mask(255, 255, 255, 255),
			},
		}); err != nil {
			return nil, errors.Wrap(err, "AddrAdd error")
		}
	}

	if err = netlink.LinkSetUp(vxlanLink); err != nil {
		return nil, errors.Wrap(err, "LinkSetUp error")
	}

	return vxlanLink, nil
}