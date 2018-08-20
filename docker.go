package main

import (
	"context"
	"fmt"
	"strings"

	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/events"
	"docker.io/go-docker/api/types/filters"
	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
)

type ipsToNamesMap map[string][]string

type dockerClienter interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(context.Context, string) (types.ContainerJSON, error)
	Events(context.Context, types.EventsOptions) (<-chan events.Message, <-chan error)
	Ping(context.Context) (types.Ping, error)
}

func getAllIPsToNames(client dockerClienter) (ipsToNamesMap, error) {
	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	allIPsToNames := make(ipsToNamesMap)

	for _, container := range containers {
		ipsToNames, err := getIPsToNames(client, container.ID)
		if err != nil {
			return nil, err
		}

		for ip, names := range ipsToNames {
			if _, ok := allIPsToNames[ip]; !ok {
				allIPsToNames[ip] = names
			} else {
				allIPsToNames[ip] = append(allIPsToNames[ip], names...)
			}
		}
	}
	return allIPsToNames, nil
}

func getIPsToNames(client dockerClienter, id string) (ipsToNamesMap, error) {
	ipsToNames := make(ipsToNamesMap)

	// ContainerList does not return all info, like Aliases
	// see: curl --unix-socket /var/run/docker.sock http://localhost/containers/json
	containerFull, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}

	for netName, netInfo := range containerFull.NetworkSettings.Networks {
		names := make([]string, 4) // 4 is worst-case size if container in a compose project (see below)

		appendNames := func(names []string, name string) []string {
			names = append(names, fmt.Sprintf("%s", name))
			names = append(names, fmt.Sprintf("%s.%s", name, netName))
			if proj, ok := containerFull.Config.Labels["com.docker.compose.project"]; ok {
				names = append(names, fmt.Sprintf("%s.%s", name, proj))
				names = append(names, fmt.Sprintf("%s.%s.%s", name, proj, netName))
			}
			return names
		}

		names = appendNames(names, strings.Trim(containerFull.Name, "/"))
		for _, name := range netInfo.Aliases {
			names = appendNames(names, name)
		}

		ipsToNames[netInfo.IPAddress] = names
	}

	return ipsToNames, nil
}

func listenForEvents(client dockerClienter) {
	eventOpts := types.EventsOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "start"),
			filters.Arg("event", "die"),
		),
	}

	events, errors := client.Events(context.Background(), eventOpts)
loop:
	for {
		select {
		case event := <-events:
			switch event.Action {
			case "start":
				ipsToNames, err := getIPsToNames(client, event.Actor.Attributes["name"])
				if err != nil {
					log.WithFields(
						log.Fields{"container": event.Actor.ID},
					).Errorf("could not get info for container: %s", err)
					continue
				}
				err = addToEtcHosts(ipsToNames)
				if err != nil {
					log.WithFields(
						log.Fields{"container": event.Actor.ID},
					).Errorf("could not add container to hosts file: %s", err)
				}
			case "die":
				// We remove by name because we cannot get the container's IP after it has been stopped.
				// Names are supposed to be kept unique by the docker deamon.
				removeFromEtcHosts(event.Actor.Attributes["name"])
			}
		case err := <-errors:
			log.Errorf("error fetching event: %s", err)
			break loop
		}
	}
}

func waitForConnection(client dockerClienter) {
	err := backoff.Retry(func() error {
		log.Info("retrying connection to docker")
		_, err := client.Ping(context.Background())
		if err != nil {
			return fmt.Errorf("error pinging docker server: %s", err)
		}
		return nil
	}, backoff.NewExponentialBackOff())
	if err != nil {
		// we should not get here with infinite backoff
		log.Fatal(err)
	}
	log.Info("connected to docker daemon")
}
