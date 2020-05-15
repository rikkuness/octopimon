package main

import "net"

// GetOutboundIP gets the preferred outbound ip of this machine
func GetOutboundIP() *net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return &localAddr.IP
}
