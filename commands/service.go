package commands

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // PPROF is loading everything in its init function
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"

	"gitlab.com/tmaczukin/hanging-droplets-cleaner/cleaner"
	"gitlab.com/tmaczukin/hanging-droplets-cleaner/version"
)

const (
	DefaultInterval int = 900
)

type ServiceCommand struct {
	provider *CleanerProvider
	cleaner  *cleaner.HangingDropletsCleaner

	listenAddr string
	interval   time.Duration
}

func (d *ServiceCommand) serveMetrics() {
	registry := prometheus.NewRegistry()
	registry.MustRegister(version.AppVersion.VersionCollector())
	registry.MustRegister(d.cleaner)
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
}

func (d *ServiceCommand) startDebugServer() error {
	if d.listenAddr == "" {
		logrus.Infoln("Metrics server disabled")
		return nil
	}

	_, _, err := net.SplitHostPort(d.listenAddr)
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		return fmt.Errorf("Invalid metrics server address: %s", err.Error())
	}

	listener, err := net.Listen("tcp", d.listenAddr)
	if err != nil {
		return err
	}

	go func() {
		logrus.Fatalln(http.Serve(listener, nil))
	}()

	d.serveMetrics()

	logrus.Infof("Metrics server listening at: %s", d.listenAddr)

	return nil
}

func (d *ServiceCommand) clean() {
	if err := d.cleaner.Clean(); err != nil {
		logrus.Fatalf("Error during cleanup: %v", err.Error())
	}
}

func (d *ServiceCommand) run() {
	d.clean()
	for {
		select {
		case <-time.After(d.interval):
			d.clean()
		}
	}
}

func (d *ServiceCommand) Execute(context *cli.Context) {
	logrus.Infoln("Running in service mode")

	d.interval = time.Duration(context.Int("interval")) * time.Second
	d.listenAddr = context.String("listen")
	d.cleaner = d.provider.GetCleaner(context)
	d.cleaner.EnableDelete()

	logrus.Infof("Droplets cleanup interval: %s", d.interval)

	if err := d.startDebugServer(); err != nil {
		logrus.Fatalln("Failed to start debug server")
	}

	d.run()
}

func NewStartCommand() *cli.Command {
	provider := &CleanerProvider{}
	cmd := &ServiceCommand{
		provider: provider,
	}

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Usage: "Debug server listen address",
			EnvVars: []string{
				"LISTEN",
			},
		},
		&cli.IntFlag{
			Name:  "interval",
			Usage: "Number of seconds between cleanup attempts",
			EnvVars: []string{
				"INTERVAL",
			},
			Value: DefaultInterval,
		},
	}
	flags = append(flags, provider.Flags()...)

	return &cli.Command{
		Name:  "service",
		Usage: "Start hanging droplets cleaner as a service mode",
		Action: func(c *cli.Context) error {
			cmd.Execute(c)
			return nil
		},
		Flags: flags,
	}
}
