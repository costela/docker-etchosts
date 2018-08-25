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

func writeToEtcHosts(ipsToNames ipsToNamesMap, config ConfigSpec) error {
	// We do not want to create the hosts file; if it's not there, we probably have the wrong path.
	// Open RW because we might have to write to it (see movePreservePerms)
	etcHosts, err := os.OpenFile(config.EtcHostsPath, os.O_RDWR, 0644)
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
	defer func(file *os.File) {
		file.Close()
		if err := os.Remove(file.Name()); err != nil && !os.IsNotExist(err) {
			log.Warnf("unexpected error trying to remove temp file %s: %s", file.Name(), err)
		}
	}(tmp)

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
				err = writeEntryWithBanner(tmp, ip, names)
				if err != nil {
					return err
				}
				delete(ipsToNames, ip) // otherwise we'll append it again below
			}
		} else {
			// keep original unmanaged line
			fmt.Fprintf(tmp, "%s\n", line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading %s: %s", config.EtcHostsPath, err)
	}

	// append remaining entries to file
	for ip, names := range ipsToNames {
		err = writeEntryWithBanner(tmp, ip, names)
		if err != nil {
			return err
		}
	}

	err = movePreservePerms(tmp, etcHosts)
	if err != nil {
		return err
	}

	return nil
}

func writeEntryWithBanner(tmp io.Writer, ip string, names []string) error {
	if len(names) > 0 {
		log.Debugf("writing entry for %s (%s)", ip, names)
		if _, err := fmt.Fprintf(tmp, "%s\n%s\t%s\n", banner, ip, strings.Join(names, " ")); err != nil {
			return fmt.Errorf("error writing entry for %s: %s", ip, err)
		}
	}
	return nil
}

func movePreservePerms(src, dst *os.File) error {
	if err := src.Sync(); err != nil {
		return fmt.Errorf("could not sync changes to %s: %s", src.Name(), err)
	}

	etcHostsInfo, err := dst.Stat()
	if err != nil {
		return fmt.Errorf("could not stat %s: %s", dst.Name(), err)
	}

	// We try moving first because it's atomic; the fallback strategy is copying the content, which might generate a
	// broken hosts file if some other process writes to it at the same time.
	err = os.Rename(src.Name(), dst.Name())
	if err != nil {
		log.Infof("could not rename to %s; falling back to less safe direct-write (%s)", dst.Name(), err)

		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := dst.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := dst.Truncate(0); err != nil {
			return err
		}

		_, err = io.Copy(dst, src)
		return err
	}

	// ensure we're not running with some umask that might break things
	err = src.Chmod(etcHostsInfo.Mode())
	if err != nil {
		return fmt.Errorf("could not chmod %s: %s", src.Name(), err)
	}
	// TODO: also keep user?

	return nil
}
