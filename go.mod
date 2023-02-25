module github.com/costela/docker-etchosts

require (
	docker.io/go-docker v1.0.0
	github.com/Microsoft/go-winio v0.4.9 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/docker/distribution v0.0.0-20170726174610-edc3ab29cdff // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/go-connections v0.3.0 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/gogo/protobuf v0.0.0-20170307180453-100ba4e88506 // indirect
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.3.0
	github.com/stretchr/testify v1.3.0 // indirect
	golang.org/x/crypto v0.0.0-20190131182504-b8fe1690c613 // indirect
	golang.org/x/net v0.0.0-20180906233101-161cd47e91fd // indirect
	golang.org/x/sys v0.1.0 // indirect
)

replace github.com/docker/docker v1.13.1 => github.com/docker/engine v0.0.0-20180816081446-320063a2ad06

go 1.16
