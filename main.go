/*
Copyright © 2018 Leo Antunes <leo@costela.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff"

	docker "docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/events"
	"docker.io/go-docker/api/types/filters"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.InfoLevel) // TODO: make configurable

	client, err := docker.NewEnvClient()
	if err != nil {
		log.Fatalf("error starting docker client: %s", err)
	}
	defer client.Close()

	var event events.Message
	eventOpts := types.EventsOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("event", "start"),
			filters.Arg("event", "die"),
		),
	}

	for {
		initialContent, err := getAllIPsToNames(client)
		if err != nil {
			log.Fatalf("error getting containers: %s", err)
		}

		log.Info("writing current content")
		syncEtcHosts(initialContent)

		events, errors := client.Events(context.Background(), eventOpts)
	innerLoop:
		for {
			select {
			case event = <-events:
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
					// We remove by name because we cannot get the container's IP after it has been destroyed.
					// Names are supposed to be kept unique by the docker deamon.}
					removeFromEtcHosts(event.Actor.Attributes["name"])
				}
			case err = <-errors:
				log.Errorf("error fetching event: %s", err)

				client.Close()
				err := backoff.Retry(func() error {
					log.Info("retrying connection to docker")
					client, err = docker.NewEnvClient()
					if err != nil {
						return fmt.Errorf("error starting docker client: %s", err)
					}
					_, err = client.Ping(context.Background())
					if err != nil {
						return fmt.Errorf("error pinging docker server: %s", err)
					}
					return nil
				}, backoff.NewExponentialBackOff())
				if err != nil {
					log.Fatal(err)
				}
				log.Info("reconnected to docker")
				defer client.Close() // client changed
				break innerLoop
			}
		}
	}
}
