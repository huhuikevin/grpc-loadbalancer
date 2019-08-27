package utils

import (
	"fmt"
	"net"
	"os"
)

func GetLocalIp() (string, error) {
	ip := os.Getenv("POD_IP")
	if ip != "" {
		fmt.Println("get pod ip = ", ip)
		return ip, nil
	}
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	fmt.Println("get host=", host)

	ips, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	fmt.Println("get ip", ips)

	return ips[0], nil
}
