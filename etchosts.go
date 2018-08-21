package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	banner = "# !!! managed by docker-etchosts !!!"
)

func writeToEtcHosts(ipsToNames ipsToNamesMap) error {
	// this will fail if the file doesn't exist, which is probably ok
	etcHosts, err := os.Open(config.EtcHostsPath)
	if err != nil {
		return fmt.Errorf("could not open %s for reading: %s", config.EtcHostsPath, err)
	}
	defer etcHosts.Close()

	// create tmpfile in same folder as
	tmp, err := ioutil.TempFile(path.Dir(config.EtcHostsPath), "docker-etchosts")
	if err != nil {
		return fmt.Errorf("could not create tempfile")
	}

	// remove tempfile; this might fail if we managed to move it, which is ok
	defer func(path string) {
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			log.Warnf("unexpected error trying to remove temp file %s: %s", path, err)
		}
	}(tmp.Name())

	// go through file and update existing entries/prune nonexistent entries
	managedLine := false
	scanner := bufio.NewScanner(etcHosts)
	for scanner.Scan() {
		line := scanner.Text()
		if line == banner {
			managedLine = true
			continue
		}
		if managedLine {
			managedLine = false
			tokens := strings.Fields(line)
			if len(tokens) < 1 {
				continue // remove empty managed line
			}
			ip := tokens[0]
			if names, ok := ipsToNames[ip]; ok {
				writeEntryWithBanner(tmp, ip, names)
				delete(ipsToNames, ip) // otherwise we'll append it again below
			}
		} else {
			// keep original unmanaged line
			fmt.Fprintf(tmp, "%s\n", line)
		}
	}

	// append remaining entries to file
	for ip, names := range ipsToNames {
		writeEntryWithBanner(tmp, ip, names)
	}

	err = movePreservePerms(tmp, etcHosts)
	if err != nil {
		return err
	}

	return nil
}

func writeEntryWithBanner(tmp io.Writer, ip string, names []string) {
	if len(names) > 0 {
		log.Infof("writing entry for %s (%s)", ip, names[0])
		fmt.Fprintf(tmp, "%s\n", banner)
		fmt.Fprintf(tmp, "%s\t%s\n", ip, strings.Join(names, " "))
	}
}

func movePreservePerms(src, dst *os.File) error {
	etcHostsInfo, err := dst.Stat()
	if err != nil {
		return fmt.Errorf("could not stat %s: %s", dst.Name(), err)
	}

	err = os.Rename(src.Name(), dst.Name())
	if err != nil {
		return fmt.Errorf("could not rename to %s: %s", config.EtcHostsPath, err)
	}

	// ensure we're not running with some umask that might break things
	err = src.Chmod(etcHostsInfo.Mode())
	if err != nil {
		return fmt.Errorf("could not chmod %s: %s", src.Name(), err)
	}
	// TODO: also keep user?

	return nil
}
