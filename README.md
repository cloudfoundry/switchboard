switchboard
===========

[![Build Status](https://travis-ci.org/cloudfoundry-incubator/switchboard.svg)](https://travis-ci.org/cloudfoundry-incubator/switchboard)

A TCP router written on Golang.

Developed to replace HAProxy as the proxy tier enabling high availability for the [MySQL dbaas for Cloud Foundry](https://github.com/cloudfoundry/cf-mysql-release). Responsible for routing of client connections to a one node at a time of a backend cluster, and failover on cluster node failure. For more information, see the develop branch of [cf-mysql-release/docs/proxy.md](https://github.com/cloudfoundry/cf-mysql-release/blob/release-candidate/docs/proxy.md).

### Why switchboard?

There are several other proxies out there: Nginx, HAProxy and even MariaDB's MaxScale. None of them met a specific criteria which is critical for the performance of the cluster in the case that a database server becomes unhealthy but is still accessible. Switchboard detects this condition (via healthchecks) and severs the connection. This forces the client to reconnect, and will be routed to a healthy backend. From the client's perspective it looks like it is connected to a single backend that briefly disappeared and is immediately available again.

## Development


### Proxy

Install **Go** by following the directions found [here](http://golang.org/doc/install)

Running the tests requires  [Ginkgo](http://onsi.github.io/ginkgo/):

```sh
go get github.com/onsi/ginkgo/ginkgo
```

Run the tests using the following command:

```sh
./bin/test
```

### UI

Ensure [phantomjs](http://phantomjs.org/) v2.0 or greater is installed.

To do this on OSX using [homebrew](http://brew.sh/):

```sh
brew install phantomjs
```

Run the UI tests using the following command:

```sh
./bin/test-ui
```

Build UI assets:

```
./bin/build-ui
```
