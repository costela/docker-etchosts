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
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	docker "github.com/docker/docker/client"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// ConfigSpec holds the runtime configuration
type ConfigSpec struct {
	LogLevel     string `default:"warn" split_words:"true"`
	EtcHostsPath string `default:"/etc/hosts" split_words:"true"`
}

var logLevelMap = map[string]log.Level{
	"debug": log.DebugLevel,
	"info":  log.InfoLevel,
	"warn":  log.WarnLevel,
	"error": log.ErrorLevel,
}

func main() {
	var config ConfigSpec

	err := envconfig.Process("etchosts", &config)
	if err != nil {
		log.Fatalf("could not parse settings from env: %s", err)
	}

	logLevel, ok := logLevelMap[strings.ToLower(config.LogLevel)]
	if !ok {
		log.Fatalf("unknown log level %s; valid values: %s", config.LogLevel, reflect.ValueOf(logLevelMap).MapKeys())
	}
	log.SetLevel(logLevel)

	quitSig := make(chan os.Signal)
	signal.Notify(quitSig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-quitSig
		cleanup(config)
	}()

	client, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		log.Fatalf("error initializing docker client: %s", err)
	}
	defer client.Close()

	for {
		waitForConnection(client)
		log.Info("listening for docker events")
		syncAndListenForEvents(client, config)
	}
}

func cleanup(config ConfigSpec) {
	log.Info("cleaning up hosts file")
	writeToEtcHosts(ipsToNamesMap{}, config)
	os.Exit(0)
}
