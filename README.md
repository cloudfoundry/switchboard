switchboard
===========

[![Build Status](https://travis-ci.org/cloudfoundry-incubator/switchboard.svg)](https://travis-ci.org/cloudfoundry-incubator/switchboard)

A TCP router written on Golang.

Developed to replace HAProxy as the proxy tier enabling high availability for the [MySQL dbaas for Cloud Foundry](https://github.com/cloudfoundry/cf-mysql-release). Responsible for routing of client connections to a one node at a time of a backend cluster, and failover on cluster node failure. For more information, see the develop branch of [cf-mysql-release/docs/proxy.md](https://github.com/cloudfoundry/cf-mysql-release/blob/release-candidate/docs/proxy.md).

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

Run the tests using the following command:

```
./bin/test-ui
```
