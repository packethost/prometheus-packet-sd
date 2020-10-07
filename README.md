A service discovery for the [Equinix Metal](https://metal.equinix.com) platform compatible with [Prometheus](https://prometheus.io).

It's based on the [Scaleway adapter](https://github.com/scaleway/prometheus-scw-sd).

## How it works

This service gets the list of servers from the Packet API and generates a file which is compatible with the Prometheus `file_sd` mechanism.

## Pre-requisites

You need your Equinix Metal Auth token. You can create the token [in the Equinix Metal web app](https://console.equinixmetal.com), click on your profile photo and navigate to "API Keys".

Make sure the Equinix Metal Auth Token is in envvar `PACKET_AUTH_TOKEN`.

## Running it

### With docker

First, export your `PACKET_AUTH_TOKEN` to an environment variable and then:

```
$ docker run -e PACKET_AUTH_TOKEN -v ${PWD}:/tmp quay.io/packet/prometheus-packet-sd --output.file=/tmp/packet.json
```

.. file `packet.json` cointaining device labels will appear in current directory.

### With the binary from Github Releases

Download the binary from the [Releases](https://github.com/packethost/prometheus-packet-sd/releases) page.


```
$ ./prometheus-packet-sd
```

.. device labels will be in `./packet.json`


If you only want to scrape a single project, you can pass project ID:

```
$ ./prometheus-packet-sd --packet.projectid 508b7fd3-0df2-4837-b0df-65aaf09f642a
```

You can also pass the project ID in envvar `PACKET_PROJECT_ID`.

The program runs a Prometheus registry itself, and reports histogram of Equinix Metal API request duration, and number of API failures. See the [main.go](main.go) for the exact label names. If you want to scrape this from a running, docker container, you should expose the port, i.e. add `-p 9465:9465` to the `docker run` command.


## Integration with Prometheus

Here is a Prometheus `scrape_config` snippet that configures Prometheus to scrape node_exporter assuming that it is deployed on all your Equinix Metal servers, and listening on port 9100.

```yaml
- job_name: node

  # Assuming that prometheus and prometheus-packet-sd are started from the same directory.
  file_sd_configs:
  - files: [ "./packet.json" ]

  # The relabeling does the following:
  # - overwrite the scrape address with the node_exporter's port.
  # - strip leading commas from the tags label.
  # - save the facility label
  # - save the instance label
  relabel_configs:
  - source_labels: [__meta_packet_public_ipv4]
    replacement: "${1}:9100"
    target_label: __address__
  - source_labels: [__meta_packet_tags]
    regex: ",(.+),"
    target_label: tags
  - source_labels: [__meta_packet_facility]
    target_label: facility
  - source_labels: [__meta_packet_hostname]
    target_label: instance

```

The following meta labels are available on targets during relabeling at the moment:

* `__meta_packet_billing_cycle`
* `__meta_packet_device_id`
* `__meta_packet_facility`
* `__meta_packet_hostname`
* `__meta_packet_operating_system`
* `__meta_packet_organization_name`
* `__meta_packet_plan`
* `__meta_packet_private_ipv4`
* `__meta_packet_project_id`
* `__meta_packet_project_name`
* `__meta_packet_public_ipv4`
* `__meta_packet_public_ipv6`
* `__meta_packet_short_id`
* `__meta_packet_state`
* `__meta_packet_switch_uuid
* `__meta_packet_tags`

## Contributing

PRs and issues are welcome.

## License

Apache License 2.0, see [LICENSE](https://github.com/packethost/prometheus-packet-sd/blob/master/LICENSE).
