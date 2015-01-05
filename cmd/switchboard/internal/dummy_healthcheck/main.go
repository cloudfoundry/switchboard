package main

import (
	"flag"

	"github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/fakes"
)

var port = flag.Uint("port", 29996, "port to listen on")

func main() {
	flag.Parse()
	fh := fakes.NewFakeHealthcheck(*port)
	fh.Start()
}
