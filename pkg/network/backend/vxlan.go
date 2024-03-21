package backend

import (
	"crypto/rand"
	"log"
	"net"
)

func NewHardwareAddr() (string, error) {

	hardwareAddr := make(net.HardwareAddr, 6)
	if _, err := rand.Read(hardwareAddr); err != nil {
		log.Printf("Failed in generating random hardware address: %s", err.Error())
		return "", err
	}

	hardwareAddr[0] = (hardwareAddr[0] & 0xfe) | 0x02
	sMac := hardwareAddr.String()
	return sMac, nil
}
