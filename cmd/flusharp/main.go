package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
)

const (
	arpCommand = "/usr/sbin/arp"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s: wrong number of arguments", os.Args[0]))
		os.Exit(1)
	}

	ip := net.ParseIP(os.Args[1])
	if ip == nil {
		fmt.Fprintf(os.Stderr, "invalid ip provided: %s", os.Args[1])
		os.Exit(1)
	}

	output, err := exec.Command(arpCommand, "-d", ip.String()).CombinedOutput()
	fmt.Printf("%s", string(output))

	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
