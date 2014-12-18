package config

type Proxy struct {
	Port                   uint
	Pidfile                string
	Backends               []Backend
	HealthcheckTimeoutInMS uint
}

type Backend struct {
	BackendIP       string
	BackendPort     uint
	HealthcheckPort uint
}
