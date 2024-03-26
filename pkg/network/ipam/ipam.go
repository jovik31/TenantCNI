package ipam

import (
	"log"
	"net/netip"
	"fmt"

	"github.com/jovik31/tenant/pkg/network/backend"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

//"errors"
//"fmt"
//"net"

// Allocate a subnet to a tenant, populate its fields if needed
//func AllocateTenant()

//func AllocateIP()
//func NewTenantIPAM() (*TenantIPAM, error) {
//	//Create a new tenant IPAM struct
//	//Return the struct and an error if any
//
//}

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

	//Generate a new hardware address for the Vxlan device
	macAddress, err := backend.NewHardwareAddr()
	if err != nil {
		log.Println("Failed to generate a new hardware address")
	}

	sMac:= macAddress.String()
	if err != nil{

		log.Println("Failed to convert hardware address to string")
	}
	//Store the Vxlan information on the tenant store
	vtepName :=fmt.Sprintf("%s.%v", tenantName, tenantVNI)
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
