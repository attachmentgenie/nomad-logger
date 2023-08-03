FROM scratch
ENTRYPOINT ["/usr/bin/nomad-logger"]
COPY nomad-logger /usr/bin/nomad-logger