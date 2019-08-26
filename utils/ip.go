package utils

import (
	"fmt"
	"net"
	"os"
)

func GetLocalIp() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	fmt.Print("get host=", host)

	ips, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	fmt.Println("get ip", ips)

	return ips[0], nil
}
