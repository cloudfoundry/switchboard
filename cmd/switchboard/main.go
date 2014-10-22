package main

import (
	"flag"
	"fmt"
	"net"
)

var (
	port = flag.Uint("port", 3306, "Port to listen on")
)

func main() {
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		fmt.Printf("Error was: %v\n", err.Error())
	}

	fmt.Printf("started on port %d\n", *port)

	for {
		l.Accept()
	}
}
