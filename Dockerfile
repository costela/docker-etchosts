FROM scratch
ENTRYPOINT ["/docker-etchosts"]
COPY docker-etchosts /
