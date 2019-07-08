A service discovery for the [Packet](https://www.packet.com/) platform compatible with [Prometheus](https://prometheus.io).

It's based on the [Scaleway adapter](https://github.com/scaleway/prometheus-scw-sd).

## How it works

This service gets the list of servers from the Packet API and generates a file which is compatible with the Prometheus `file_sd` mechanism.

## Pre-requisites

You need your Packet Auth token. You can create the token [in the Packet web app](https://app.packet.net), click on your profile photo and navigate to "API Keys".

## Installing it

Download the binary from the [Releases](https://github.com/packethost/prometheus-packet-sd/releases) page.

## Running it
host
```
usage: sd adapter usage --packet.token-file=my-token.txt [<flags>]

Tool to generate Prometheus file_sd target files for Packet.

Flags:
  -h, --help                    Show context-sensitive help (also try --help-long and --help-man).
      --output.file="packet.json"  The output filename for file_sd compatible file.
      --packet.organization=SCW.ORGANIZATION
                                The Packet organization.
      --packet.region="par1"       The Packet region. Leaving blank will fetch from all the regions.
      --packet.token-file=""       The authentication token file containing Packet Secret Key.
      --target.refresh=30       The refresh interval (in seconds).
      --target.port=80          The default port number for targets.
      --web.listen-address=":9465"
                                The listen address.
      --version                 Show application version.
```

## Integration with Prometheus

Here is a Prometheus `scrape_config` snippet that configures Prometheus to scrape node_exporter assuming that it is deployed on all your Packet servers.

```yaml
- job_name: node

  # Assuming that prometheus and prometheus-packet-sd are started from the same directory.
  file_sd_configs:
  - files: [ "./packet.json" ]

  # The relabeling does the following:
  # - overwrite the scrape address with the node_exporter's port.
  # - strip leading commas from the tags label.
  # - save the region label (par1/ams1).
  # - overwrite the instance label with the server's name.
  relabel_configs:
  - source_labels: [__meta_packet_private_ip]
    replacement: "${1}:9100"
    target_label: __address__
  - source_labels: [__meta_packet_tags]
    regex: ",(.+),"
    target_label: tags
  - source_labels: [__meta_packet_location_zone_id]
    target_label: region
  - source_labels: [__meta_packet_name]
    target_label: instance
```

The following meta labels are available on targets during relabeling:

* `__meta_packet_architecture`: the architecture of the server.
* `__meta_packet_blade_id`: the identifier of the blade (can be empty).
* `__meta_packet_chassis_id`: the identifier of the chassis (can be empty).
* `__meta_packet_cluster_id`: the identifier of the cluster (can be empty).
* `__meta_packet_commercial_type`: the commercial type of the server (eg START1-XS).
* `__meta_packet_hypervisor_id`: the identifier of the hypervisor.
* `__meta_packet_identifier`: the identifier of the server.
* `__meta_packet_image_id`: the identifier of the server's image.
* `__meta_packet_image_name`: the name of the server's image.
* `__meta_packet_name`: the name of the server.
* `__meta_packet_node_id`: the identifier of the node.
* `__meta_packet_organization`: the organization owning the server.
* `__meta_packet_platform_id`: the identifier of the platform.
* `__meta_packet_private_ip`: the private IP address of the server.
* `__meta_packet_public_ip`: the public IP address of the server (can be empty).
* `__meta_packet_state`: the state of the server.
* `__meta_packet_tags`: comma-separated list of tags associated to the server (trailing commas on both sides).
* `__meta_packet_zone_id`: the identifier of the zone (region).


## Contributing

PRs and issues are welcome.

## License

Apache License 2.0, see [LICENSE](https://github.com/packethost/prometheus-packet-sd/blob/master/LICENSE).
