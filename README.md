A service discovery for the [Packet](https://www.packet.com/) platform compatible with [Prometheus](https://prometheus.io).

It's based on the [Scaleway adapter](https://github.com/scaleway/prometheus-scw-sd).

## How it works

This service gets the list of servers from the Packet API and generates a file which is compatible with the Prometheus `file_sd` mechanism.

## Pre-requisites

You need your Packet Auth token. You can create the token [in the Packet web app](https://app.packet.net), click on your profile photo and navigate to "API Keys".

## Installing it

Download the binary from the [Releases](https://github.com/packethost/prometheus-packet-sd/releases) page.

## Running it

```
> ./prometheus-packet-sd --help
usage: sd adapter usage --packet.projectid=PACKET.PROJECTID --packet.authtoken=PACKET.AUTHTOKEN [<flags>]

Tool to generate Prometheus file_sd target files for Packet.

Flags:
  -h, --help               Show context-sensitive help (also try --help-long and
                           --help-man).
      --output.file="packet.json"  
                           The output filename for file_sd compatible file.
      --packet.projectid=PACKET.PROJECTID  
                           Packet project ID.
      --packet.authtoken=PACKET.AUTHTOKEN  
                           Packet auth token.
      --target.refresh=30  The refresh interval (in seconds).
      --target.port=9100   The default port number for targets.
      --web.listen-address=":9465"  
                           The listen address.
      --version            Show application version.
```


## Integration with Prometheus

Here is a Prometheus `scrape_config` snippet that configures Prometheus to scrape node_exporter assuming that it is deployed on all your Packet servers.

TODO: update this section

```yaml
- job_name: node

  # Assuming that prometheus and prometheus-packet-sd are started from the same directory.
  file_sd_configs:
  - files: [ "./packet.json" ]

  # The relabeling does the following:
  # - overwrite the scrape address with the node_exporter's port.
  # - strip leading commas from the tags label.
  # - save the facility label
  # - save the hostname label
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
    target_label: hostname

```

The following meta labels are available on targets during relabeling at the moment:

* `__meta_packet_billing_cycle`
* `__meta_packet_facility`
* `__meta_packet_hostname`
* `__meta_packet_plan`
* `__meta_packet_private_ipv4`
* `__meta_packet_public_ipv4`
* `__meta_packet_public_ipv6`
* `__meta_packet_state`
* `__meta_packet_tags`



## Contributing

PRs and issues are welcome.

## License

Apache License 2.0, see [LICENSE](https://github.com/packethost/prometheus-packet-sd/blob/master/LICENSE).
