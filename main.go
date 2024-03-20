package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	promcolversion "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promversion "github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"

	"github.com/attachmentgenie/nomad-logger/fluentbit"
	"github.com/attachmentgenie/nomad-logger/nomad"
	"github.com/attachmentgenie/nomad-logger/promtail"
)

var args struct {
	NomadAddress        string `arg:"--nomad-addr,env:NOMAD_ADDR" default:"http://localhost:4646" help:"The address of the Nomad API"`
	NomadAllocsDir      string `arg:"--nomad-allocs-dir,env:NOMAD_ALLOCS_DIR" default:"/var/lib/nomad/alloc" help:"The location of the Nomad allocations data. Used to set the path to the logfiles"`
	NomadNodeID         string `arg:"--nomad-node-id,env:NOMAD_NODE_ID" default:"" help:"The ID of the Nomad node to collect logs for. If empty, we'll suppose this also runs in as a nomad job, and the available env vars will be used to determine the Node ID"`
	NomadMetaPrefix     string `arg:"--nomad-meta-prefix,env:NOMAD_META_PREFIX" default:"nomad-logger" help:"Consider meta keys that start with '$prefix.'. See log shippers for more info on meta usage."`
	ReloadCmd           string `arg:"--reload-cmd,env:RELOAD_CMD" default:"" help:"Optional command to execute after logshipper config has changed. Usefull to signal a service to reload it's config. Valid for fluentbit logshipper."`
	LogShipper          string `arg:"--log-shipper,env:LOG_SHIPPER" default:"promtail" help:"The logshipper to use. Options: fluentbit, promtail"`
	FluentbitConfFile   string `arg:"--fluentbit-conf-file,env:FLUENTBIT_CONF_FILE" default:"/etc/fluent-bit/nomad.conf" help:"The file in which we can write our input's and stuff. Will be completely overwritten, should be '@INCLUDE'ed from main config file."`
	FluentbitTagPrefix  string `arg:"--fluentbit-tag-prefix,env:FLUENTBIT_TAG_PREFIX" default:"nomad" help:"Prefix to use for fluentbit tags. Full tag will be '$prefix.$allocId"`
	FluentbitParser     string `arg:"--fluentbit-parser,env:FLUENTBIT_PARSER" default:"" help:"Parser to apply on every input. Empty string for none"`
	PromtailTargetsFile string `arg:"--promtail-targets-file,env:PROMTAIL_TARGETS_FILE" default:"/etc/promtail/nomad.yaml" help:"The promtail file_sd_config file where the generated config can be written. Will be completely overwritten, so don't put anything else there."`
	PrometheusPort      string `arg:"--metrics-port,env:METRICS_PORT" default:"2112" help:"The port to show metrics on"`
	CheckInterval       int64  `arg:"--check-interval,env:CHECK_INTERVAL" default:"1" help:"Interval (sec) between checking for new allocations."`
}

var (
	commit  = "none"
	date    = "unknown"
	service = "nomad-logger"
	version = "dev"
)

func main() {
	promversion.Revision = commit
	promversion.BuildDate = date
	promversion.Version = version
	promNamespace := strings.ReplaceAll(string(service), "-", "_")

	arg.MustParse(&args)

	nmd := &nomad.Nomad{
		Address:    args.NomadAddress,
		AllocsDir:  args.NomadAllocsDir,
		MetaPrefix: args.NomadMetaPrefix,
	}

	err := nmd.NewClient()
	if err != nil {
		log.Fatalln(err)
	}

	if args.NomadNodeID != "" {
		nmd.NodeID = args.NomadNodeID
	} else {
		nmdErr := nmd.SetNodeIDFromEnvs()
		if nmdErr != nil {
			log.Fatalf("no nomad id found '%s'", nmdErr.Error())
		}
	}
	slog.Info("collecting allocs for", "node_id", nmd.NodeID)

	reg := prometheus.NewRegistry()
	m := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: promNamespace,
		Name:      "allocs_processed",
	})
	reg.MustRegister(m)
	switch args.LogShipper {
	case "fluentbit":
		slog.Info("Starting nomad-logger for Fluentbit")
		fb := &fluentbit.Fluentbit{
			Nomad:         nmd,
			ConfFile:      args.FluentbitConfFile,
			TagPrefix:     args.FluentbitTagPrefix,
			Parser:        args.FluentbitParser,
			ReloadCmd:     args.ReloadCmd,
			CheckInterval: args.CheckInterval,
		}
		go fb.Run(m)
	case "promtail":
		slog.Info("Starting nomad-logger for Promtail")
		pt := &promtail.Promtail{
			Nomad:         nmd,
			TargetsFile:   args.PromtailTargetsFile,
			CheckInterval: args.CheckInterval,
		}
		go pt.Run(m)
	default:
		log.Fatalf("Invalid log shipper type '%s'", args.LogShipper)
	}

	reg.MustRegister(promcolversion.NewCollector(promNamespace))
	reg.Unregister(collectors.NewGoCollector())

	landingConfig := web.LandingConfig{
		Name:    service,
		Version: version,
		Links: []web.LandingLinks{
			{
				Address: "/metrics",
				Text:    "metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		panic(err)
	}

	http.Handle("/", landingPage)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	httpErr := http.ListenAndServe(fmt.Sprintf(":%s", args.PrometheusPort), nil)
	if httpErr != nil {
		log.Fatalln(httpErr.Error())
	}
}
