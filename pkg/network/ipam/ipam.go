package ipam

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"errors"
	//"golang.org/x/exp/maps"

	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	cip "github.com/containernetworking/plugins/pkg/ip"
)

var (
	ErrIPOverflow = errors.New(" ip overflow")
)

func NewNodeIPAM(store *NodeStore, nodeName string) (*NodeIPAM, error) {

	nim := &NodeIPAM{
		NodeName:  nodeName,
		NodeStore: store,
	}

	return nim, nil
}

func NewTenantIPAM(store *TenantStore, tenantName string) (*TenantIPAM, error) {

	tim := &TenantIPAM{
		TenantName:  tenantName,
		TenantStore: store,
	}
	return tim, nil
}

func NewPodIPAM(store *PodStore) (*PodIPAM, error) {

	pim := &PodIPAM{
		PodStore: store,
	}
	return pim, nil
}

func ListSubnets(original string, newPrefix int) []string {

	var subnetList []string
	subnet := ipaddr.NewIPAddressString(original).GetAddress()

	iterator := subnet.SetPrefixLen(newPrefix).PrefixIterator()
	for iterator.HasNext() {
		subnetList = append(subnetList, iterator.Next().String())
	}
	return subnetList
}

// Check if it is possible to create a tenantStore outside and pass it to the function only updating the tenantCIDR
func (nim *NodeIPAM) AllocateTenant(tenantName string, tenantVNI int) error {
	nim.NodeStore.Lock()
	defer nim.NodeStore.Unlock()

	if err := nim.NodeStore.LoadNodeData(); err != nil {

		return err
	}

	availableList := nim.NodeStore.Data.AvailableList
	if len(availableList) <= 0 {
		log.Println("No more available subnets for tenants in this node")
		return nil
	}

	tenantCIDR, err := netip.ParsePrefix(availableList[0])
	if err != nil {

		log.Printf("Failed parsing tenant CIDR prefix from available list")
	}

	//Update values for available subnet slice and for tenants map
	nim.NodeStore.Data.AvailableList = availableList[1:]
	nim.NodeStore.Data.TenantList[tenantName] = tenantCIDR
	nim.NodeStore.StoreNodeData()

	tenantStore, err := NewTenantStore(defaultStoreDir, tenantName)
	if err != nil {
		log.Printf("Failed to create a tenant Store")
	}

	tenantStore.Data.TenantCIDR = tenantCIDR.String()

	//Generate a new bridge name for the tenant
	tenantStore.Data.Bridge = &Bridge{
		Name:    "br-" + tenantName,
		Gateway: tenantCIDR.Addr().Next(),
	}
	log.Printf("Bridge name: %s and IP: %s", tenantStore.Data.Bridge.Name, tenantStore.Data.Bridge.Gateway.String())
	tenantStore.Data.Last = tenantStore.Data.Bridge.Gateway.String()


	if len(tenantStore.Data.Bridge.Name) >= 13 {
		log.Printf("Bridge name too long: %s", tenantStore.Data.Bridge.Name)
		tenantStore.Data.Bridge.Name = "br-default"
	}

	//Create Bridge
	_, err = backend.CreateTenantBridge(tenantStore.Data.Bridge.Name, 1450, tenantStore.Data.Bridge.Gateway)
	if err != nil {
		log.Printf("Failed creating %s bridge", err)
	}

	//Generate a new hardware address for the Vxlan device
	macAddress, err := backend.NewHardwareAddr()
	if err != nil {
		log.Println("Failed to generate a new hardware address")
	}

	sMac := macAddress.String()
	if err != nil {

		log.Println("Failed to convert hardware address to string")
	}
	//Store the Vxlan information on the tenant store
	vtepName := fmt.Sprintf("%s.%v", tenantName, tenantVNI)
	log.Println(vtepName)
	tenantStore.Data.Vxlan = &Vxlan{
		VtepName: vtepName,
		VtepIP:   tenantCIDR.Addr().String(),
		VtepMac:  sMac,
		VNI:      tenantVNI,
	}

	return tenantStore.StoreTenantData()

}

func GetTenantIP(tenantList []string) netip.Prefix {

	subnet, err := netip.ParsePrefix(tenantList[0])
	if err != nil {
		log.Println("Error parsing tenant subnet: ", err)
	}
	return subnet
}

func (tim *TenantIPAM) AllocateIP(id string, ifName string) (net.IP, error) {

	tim.TenantStore.Lock()
	defer tim.TenantStore.Unlock()

	if err := tim.TenantStore.LoadTenantData(); err != nil {
		log.Println("Failed to load tenant data")
	}
	//Get tenant gateway
	gtw := tim.TenantStore.Data.Bridge.Gateway.String()
	
	//Check if ID already exists
	ip, _ := tim.TenantStore.GetIPByID(id)
	if len(ip) > 0 {
		log.Println("ID already exists")
		return ip, nil
	}
	lastIP := tim.TenantStore.Last()
	if len(lastIP) == 0 {
		lastIP = net.IP(gtw)
	}
	start := make(net.IP, len(lastIP))
	copy(start, lastIP)
	log.Printf("Last IP: %s, Start IP %s and gateway is: %s", lastIP.String(), start.String(), gtw)
	for {
		next, err := tim.NextIP(start)
		if err == ErrIPOverflow && !lastIP.Equal(net.IP(gtw)){
			start = net.IP(gtw)
			continue
		} else if err != nil {
			log.Println("Error getting next IP: ", err)
			return nil, err
		}
		if !tim.TenantStore.Contains(next) {
			err := tim.TenantStore.Add(next, id, ifName)
			tim.TenantStore.Data.Last = next.String()
			tim.TenantStore.StoreTenantData()
			return next, err
		}
		start = next
		if start.Equal(lastIP) {
			break
		}
		log.Printf("Next IP: %s", next.String())
	}
	return nil, fmt.Errorf("no more available IPs")
	
}


func (tim *TenantIPAM)NextIP(ip net.IP) (net.IP, error) {

	next := cip.NextIP(ip)
	log.Println("Next IP: ", next.String())

	subnet := tim.TenantStore.Data.TenantCIDR
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Printf("Failed to parse CIDR: %s", err)
		return nil, err
	}
	if !ipnet.Contains(next) {
		log.Println("IP overflow")
		return nil, ErrIPOverflow
	}
	return next, nil
}
func (tim *TenantIPAM) IPNet(ip net.IP) *net.IPNet {
	
	_, ipNet, err := net.ParseCIDR(tim.TenantStore.Data.TenantCIDR)
	if err != nil {
		log.Printf("Failed to parse CIDR: %s", err)
	}
	return &net.IPNet{IP: ip, Mask: ipNet.Mask}
}