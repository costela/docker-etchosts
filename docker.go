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
		names := make([]string, 0, 4) // 4 is worst-case size if container in a compose project (see below)

		appendNames := func(names []string, name string) []string {
			log.Debugf("found base name %s with IP %s", name, netInfo.IPAddress)
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

func syncAndListenForEvents(client dockerClienter) {

	eventOpts := types.EventsOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "start"),
			filters.Arg("event", "destroy"),
		),
	}

	// helper channel to ensure we run once without
	kickoff := make(chan bool, 1)
	kickoff <- true
	defer close(kickoff)

	events, errors := client.Events(context.Background(), eventOpts)
loop:
	for {
		select {
		case <-kickoff:
			log.Infof("running initial sync")
			getAndWrite(client)
		case event := <-events:
			log.Infof("got %s event for %s", event.Action, event.Actor.Attributes["name"])
			getAndWrite(client)
		case err := <-errors:
			log.Errorf("error fetching event: %s", err)
			break loop
		}
	}
}

func getAndWrite(client dockerClienter) {
	log.Info("fetching container infos")
	currentContent, err := getAllIPsToNames(client)
	if err != nil {
		log.Errorf("error getting container infos: %s", err)
	}

	log.Info("writing current state")
	err = writeToEtcHosts(currentContent)
	if err != nil {
		log.Errorf("error syncing hosts: %s", err)
	}
}

func waitForConnection(client dockerClienter) {
	err := backoff.Retry(func() error {
		log.Info("retrying connection to docker")
		_, err := client.Ping(context.Background())
		if err != nil {
			log.Errorf("error pinging docker server: %s", err)
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
