[![Build Status](https://travis-ci.org/costela/docker-etchosts.svg?branch=master)](https://travis-ci.org/costela/docker-etchosts)
[![Go Report Card](https://goreportcard.com/badge/github.com/costela/docker-etchosts)](https://goreportcard.com/report/github.com/costela/docker-etchosts)

# docker-etchosts

Automatically manages entries in hosts file (`/etc/hosts`) for local [docker](https://docker.io/) containers.

Its main use-case is working on multiple web-accessible projects without having to keep track of different exported ports, instead relying on predictable names.

Inspired by [`vagrant-hostmanager`](https://github.com/devopsgroup-io/vagrant-hostmanager) and [`docker-hostmanager`](https://github.com/iamluc/docker-hostmanager).

## Installation

To install from source, go >= 1.13 is required:
```
go get -d github.com/costela/docker-etchosts
cd $GOPATH/src/github.com/costela/docker-etchosts
go install
```
And run it as `docker-etchosts`.  
(using `go get` directly is currently not possible because of an issue between `docker.io/go-docker` and `go mod`)

Alternatively, it's also possible to run `docker-etchost` from inside a docker container itself, giving it access to both the hosts file and the docker daemon:
```
docker run -d \
  --network none --restart always \
  -v /etc/hosts:/etc/hosts -v /var/run/docker.sock:/var/run/docker.sock \
  costela/docker-etchosts
```

## Usage

Once started, `docker-etchosts` creates `/etc/hosts` entries for all existing containers with accessible networks. It also listens for events from the docker deamon, updating the hosts file for each container created or destroyed.

Entries are created for each container network with the following names:
- container name plus all network-specific aliases
- (optionally) each of the above with the [docker-compose](https://github.com/docker/compose) project name appended
- each of the above with the network name appended (except for the default `bridge` network)

Each container will thereforr have up to 4 entries per alias: CONTAINER_ALIAS, CONTAINER_ALIAS.PROJECT, CONTAINER_ALIAS.NETWORK_NAME, CONTAINER_ALIAS.PROJECT.NETWORK_NAME

Arbitrary hosts entries can be added via a custom label (`com.costela.docker-etchosts.add_hosts`) by specifying a single or array of host names.

This means the following `docker-compose.yml` setup for project `someproject`:
```yaml
services:
  someservice:
    ...
    labels:
        - 'com.costela.docker-etchosts.add_hosts=["a.example.com", "b.example.com"]'
    networks:
      somenet:
        aliases:
          - somealias
```
Would generate the following hosts entry:
```
x.x.x.x     someservice someservice.somenet someservice.someproject someservice.someproject.somenet somealias somealias.somenet somealias.someproject somealias.someproject.somenet a.example.com b.example.com
```

_NOTE_: Docker ensures the uniqueness of containers' IP addresses and names, but does not ensure uniqueness for aliases. This may lead to multiple entries having the same name, especially for the shorter name versions. The longer, more explict, names are there to help in these cases, enabling different workflows with multiple projects.

To avoid overwriting unrelated entries, `docker-etchosts` will not touch entries not managed by itself. If you already manually created hosts entries for IPs used by containers, you should remove them so that `docker-etchosts` can take over management.

All entries managed by `docker-etchosts` will be removed upon termination, returning the hosts file to its initial state.

## Configuration

`docker-etchosts` can be configured with the following environment variables:

- **`ETCHOSTS_LOG_LEVEL`**: set the verbosity of log messages (default: `warn`, possible values: `debug` `info` `warn` `error`)

- **`ETCHOSTS_ETC_HOSTS_PATH`**: path to hosts file (default `/etc/hosts`)
