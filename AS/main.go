package main

import (
	"fmt"
	"net"
	"strings"
)

var dnsRecords = make(map[string]string)

func main() {
	addr := net.UDPAddr{
		Port: 53333,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Authoritative Server running on UDP port 53333")

	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		message := string(buffer[:n])
		fmt.Println("Received:\n", message)

		response := handleMessage(message)

		conn.WriteToUDP([]byte(response), clientAddr)
	}
}

func handleMessage(msg string) string {
	lines := strings.Split(msg, "\n")
	var name, value string

	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			name = strings.TrimPrefix(line, "NAME=")
		}
		if strings.HasPrefix(line, "VALUE=") {
			value = strings.TrimPrefix(line, "VALUE=")
		}
	}

	// Registration
	if value != "" {
		dnsRecords[name] = value
		fmt.Println("Registered:", name, "->", value)
		return "Registered\n"
	}

	// Query
	ip := dnsRecords[name]
	return fmt.Sprintf("TYPE=A\nNAME=%s\nVALUE=%s\nTTL=10\n", name, ip)
}
