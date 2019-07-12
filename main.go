// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/packethost/packngo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

var (
	a         = kingpin.New("sd adapter usage", "Tool to generate Prometheus file_sd target files for Packet.")
	outputf   = a.Flag("output.file", "The output filename for file_sd compatible file.").Default("packet.json").String()
	projectid = a.Flag("packet.projectid", "Packet project ID.").String()
	token     = a.Flag("packet.authtoken", "Packet auth token.").Envar("PACKET_AUTH_TOKEN").Required().String()
	refresh   = a.Flag("target.refresh", "The refresh interval (in seconds).").Default("30").Int()
	port      = a.Flag("target.port", "The default port number for targets.").Default("9100").Int()
	listen    = a.Flag("web.listen-address", "The listen address.").Default(":9465").String()

	packetPrefix = model.MetaLabelPrefix + "packet_"
)

var (
	reg             = prometheus.NewRegistry()
	requestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "prometheus_packet_sd_request_duration_seconds",
			Help:    "Histogram of latencies for requests to the Packet API.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
	)
	requestFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_packet_sd_request_failures_total",
			Help: "Total number of failed requests to the Packet API.",
		},
	)
)

func init() {
	reg.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(version.NewCollector("prometheus_packet_sd"))
	reg.MustRegister(requestDuration)
	reg.MustRegister(requestFailures)
}

type packetLogger struct {
	log.Logger
}

// LogHTTP implements the Logger interface of the Packet API.
func (l *packetLogger) LogHTTP(r *http.Request) {
	level.Debug(l).Log("msg", "HTTP request", "method", r.Method, "url", r.URL.String())
}

// Fatalf implements the Logger interface of the Packet API.
func (l *packetLogger) Fatalf(format string, v ...interface{}) {
	level.Error(l).Log("msg", fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Debugf implements the Logger interface of the Packet API.
func (l *packetLogger) Debugf(format string, v ...interface{}) {
	level.Debug(l).Log("msg", fmt.Sprintf(format, v...))
}

// Infof implements the Logger interface of the Packet API.
func (l *packetLogger) Infof(format string, v ...interface{}) {
	level.Info(l).Log("msg", fmt.Sprintf(format, v...))
}

// Warnf implements the Logger interface of the Packet API.
func (l *packetLogger) Warnf(format string, v ...interface{}) {
	level.Warn(l).Log("msg", fmt.Sprintf(format, v...))
}

// Warnf implements the Logger interface of the promhttp package.
func (l *packetLogger) Println(v ...interface{}) {
	level.Error(l).Log("msg", fmt.Sprintln(v...))
}

// packetDiscoverer retrieves target information from the Packet API.
type packetDiscoverer struct {
	client    *packngo.Client
	port      int
	refresh   int
	separator string
	lasts     map[string]struct{}
	logger    log.Logger
}

func labelName(postfix string) string {
	return packetPrefix + postfix
}

func (d *packetDiscoverer) createTarget(device *packngo.Device) *targetgroup.Group {
	var tags string
	if len(device.Tags) > 0 {
		tags = d.separator + strings.Join(device.Tags, d.separator) + d.separator
	}
	networkInfo := device.GetNetworkInfo()

	addr := net.JoinHostPort(networkInfo.PrivateIPv4, fmt.Sprintf("%d", d.port))

	return &targetgroup.Group{
		Source: fmt.Sprintf("packet/%s", device.ID),
		Targets: []model.LabelSet{
			model.LabelSet{
				model.AddressLabel: model.LabelValue(addr),
			},
		},
		Labels: model.LabelSet{
			model.AddressLabel: model.LabelValue(addr),

			model.LabelName(labelName("hostname")):      model.LabelValue(device.Hostname),
			model.LabelName(labelName("state")):         model.LabelValue(device.State),
			model.LabelName(labelName("billing_cycle")): model.LabelValue(device.BillingCycle),
			model.LabelName(labelName("plan")):          model.LabelValue(device.Plan.Slug),
			model.LabelName(labelName("facility")):      model.LabelValue(device.Facility.Code),
			model.LabelName(labelName("private_ipv4")):  model.LabelValue(networkInfo.PrivateIPv4),
			model.LabelName(labelName("public_ipv4")):   model.LabelValue(networkInfo.PublicIPv4),
			model.LabelName(labelName("public_ipv6")):   model.LabelValue(networkInfo.PublicIPv6),
			model.LabelName(labelName("tags")):          model.LabelValue(tags),
			model.LabelName(labelName("project_id")):    model.LabelValue(device.Project.ID),
		},
	}
}

func (d *packetDiscoverer) getTargets() ([]*targetgroup.Group, error) {
	now := time.Now()
	devices := []packngo.Device{}
	if *projectid == "" {
		projects, _, err := d.client.Projects.List(nil)
		requestDuration.Observe(time.Since(now).Seconds())

		if err != nil {
			requestFailures.Inc()
			return nil, err
		}
		for _, p := range projects {
			now = time.Now()
			ds, _, err := d.client.Devices.List(p.ID, nil)
			requestDuration.Observe(time.Since(now).Seconds())
			if err != nil {
				requestFailures.Inc()
				return nil, err
			}
			devices = append(devices, ds...)
		}

	} else {
		ds, _, err := d.client.Devices.List(*projectid, nil)
		requestDuration.Observe(time.Since(now).Seconds())
		if err != nil {
			requestFailures.Inc()
			return nil, err
		}
		devices = ds
	}

	level.Debug(d.logger).Log("msg", "get devices", "nb", len(devices))

	current := make(map[string]struct{})
	tgs := make([]*targetgroup.Group, len(devices))
	for _, device := range devices {
		tg := d.createTarget(&device)
		level.Debug(d.logger).Log("msg", "device added", "source", tg.Source)
		current[tg.Source] = struct{}{}
		tgs = append(tgs, tg)
	}

	// Add empty groups for devices which have been removed since the last refresh.
	for k := range d.lasts {
		if _, ok := current[k]; !ok {
			level.Debug(d.logger).Log("msg", "device deleted", "source", k)
			tgs = append(tgs, &targetgroup.Group{Source: k})
		}
	}
	d.lasts = current

	return tgs, nil
}

func (d *packetDiscoverer) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refresh) * time.Second); ; {
		tgs, err := d.getTargets()
		if err == nil {
			ch <- tgs
		}

		// Wait for ticker or exit when ctx is closed.
		select {
		case <-c:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	a.HelpFlag.Short('h')

	a.Version(version.Print("prometheus-packet-sd"))

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger := &packetLogger{
		log.With(
			log.NewSyncLogger(log.NewLogfmtLogger(os.Stdout)),
			"ts", log.DefaultTimestampUTC,
			"caller", log.DefaultCaller,
		),
	}

	client := packngo.NewClientWithAuth("prometheus_sd", *token, nil)

	_, _, err = client.Projects.List(nil)
	if err != nil {
		fmt.Println("failed to check Packet credentials:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	disc := &packetDiscoverer{
		client:    client,
		port:      *port,
		refresh:   *refresh,
		separator: ",",
		logger:    logger,
		lasts:     make(map[string]struct{}),
	}
	sdAdapter := NewAdapter(ctx, *outputf, "packetSD", disc, logger)
	sdAdapter.Run()

	level.Debug(logger).Log("msg", "listening for connections", "addr", *listen)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorLog: logger}))
	if err := http.ListenAndServe(*listen, nil); err != nil {
		level.Debug(logger).Log("msg", "failed to listen", "addr", *listen, "err", err)
		os.Exit(1)
	}
}
