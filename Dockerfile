FROM gcr.io/distroless/base
COPY prometheus-packet-sd /prometheus-packet-sd
ENTRYPOINT ["/prometheus-packet-sd"]
