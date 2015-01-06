package main

import (
	"flag"

	"github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/fakes"
	"github.com/tedsuo/ifrit"
)

var port = flag.Uint("port", 19996, "port to listen on")
var healthcheckPort = flag.Uint("healthcheckPort", 29996, "port healthcheck will listen on")

func main() {
	flag.Parse()
	fb := fakes.NewFakeBackend(*port, *healthcheckPort)
	fbProcess := ifrit.Invoke(fb)
	<-fbProcess.Wait()
}
