# docker-etchosts

Automatically manages entries in `/etc/hosts` for local [docker](https://docker.io/) containers.

# Usage

To enable 

_NOTE_: to avoid overwriting unrelated settings, `docker-etchosts` will not touch entries not created by itself. If
you already manually created hosts entries for containers, you should remove them so that `docker-etchosts` can take
over management.

# How it works

