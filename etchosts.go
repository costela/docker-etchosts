package main

import (
	"fmt"
	"strings"
)

/*
TODO:
- check mtime one save, warn if changed and skip saving
- re-read on every save
- inotify + resync on change ?
*/

const (
	etcHostsPath = "/etc/hosts" // TODO: make configurable via env
)

func syncEtcHosts(initialContent ipsToNamesMap) error {
	for ip, names := range initialContent {
		fmt.Printf("%s\t%s\n", ip, strings.Join(names, " "))
	}

	return nil
}

func addToEtcHosts(ipsToNames ipsToNamesMap) error {
	for ip, names := range ipsToNames {
		fmt.Printf("%s\t%s\n", ip, strings.Join(names, " "))
	}

	return nil
}

func removeFromEtcHosts(name string) error {
	fmt.Printf("removing %s\n", name)
	return nil
}
