package main

import (
	"context"
	"fmt"
	"strings"

	"docker.io/go-docker/api/types"
)

type ipsToNamesMap map[string][]string

type dockerClienter interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(context.Context, string) (types.ContainerJSON, error)
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
