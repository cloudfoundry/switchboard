package main

import (
	"flag"

	"github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/fakes"
	"github.com/tedsuo/ifrit"
)

var port = flag.Uint("port", 29996, "port to listen on")

func main() {
	flag.Parse()
	fh := fakes.NewFakeHealthcheck(*port)
	fhProcess := ifrit.Invoke(fh)
	<-fhProcess.Wait()
}
