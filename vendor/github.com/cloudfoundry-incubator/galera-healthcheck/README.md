galera-healthcheck
==================

**Note:** This project is not intended to stand alone. It is a supporting project that is used with the Cloud Foundry project, [cf-mysql-release](https://github.com/cloudfoundry/cf-mysql-release). While this project is open source (so you can fork it and do whatever you like within the requirements of the license), the community will likely accept only PRs that are not in conflict with the intended purpose of the project.

This go-based process is designed to run on a MariaDB Galera node and monitor the health of the node.
An http endpoint is opened, by default at '/' on port 9200.
A healthy node will return HTTP status 200, and a node that should not be accessed returns a 503.

Several commandline flags are supported, run `galera-healthcheck -h` for more information.
  * More information about the config string can be found in the documentation of the general configuration library  [service-config](https://github.com/pivotal-cf-experimental/service-config).

##Running tests##
Run `./bin/test` for unit tests. Running tests using `ginkgo` will not work because a config file is necessary. 
