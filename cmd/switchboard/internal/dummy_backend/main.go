package main

import (
	"flag"

	"github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/fakes"
)

var port = flag.Uint("port", 19996, "port to listen on")

func main() {
	flag.Parse()
	fb := fakes.NewFakeBackend(*port)
	fb.Start()
}
