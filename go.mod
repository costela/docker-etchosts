module github.com/costela/docker-etchosts

require (
	docker.io/go-docker v1.0.0
	github.com/Microsoft/go-winio v0.4.9 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/gogo/protobuf v0.0.0-20170307180453-100ba4e88506 // indirect
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/stretchr/testify v1.3.0 // indirect
	golang.org/x/net v0.7.0 // indirect
)

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

go 1.16
