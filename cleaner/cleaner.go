package cleaner

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/digitalocean/godo"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/tmaczukin/hanging-droplets-cleaner/client"
)

var (
	numberOfRemovedDroplets = prometheus.NewDesc(
		"hanging_droplets_cleaner_remove_droplets_total",
		"Total number of removed droplets",
		[]string{},
		nil,
	)

	numberOfStopDropletErrors = prometheus.NewDesc(
		"hanging_droplets_cleaner_stop_droplet_errors_total",
		"Total number of droplets stopping errors",
		[]string{},
		nil,
	)

	numberOfRemoveDropletErrors = prometheus.NewDesc(
		"hanging_droplets_cleaner_remove_droplet_errors_total",
		"Total number of droplets removing errors",
		[]string{},
		nil,
	)
)

type HangingDropletsCleaner struct {
	client         client.DigitalOceanClientInterface
	machinesFinder MachinesFinderInterface

	delete             bool
	runnerPrefix       []string
	runnerPrefixRegexp *regexp.Regexp
	dropletAge         time.Duration

	totalNumberOfRemovedDroplets     int64
	totalNumberOfStopDropletErrors   int64
	totalNumberOfRemoveDropletErrors int64
}

func (c *HangingDropletsCleaner) Describe(ch chan<- *prometheus.Desc) {
	ch <- numberOfRemovedDroplets
	ch <- numberOfStopDropletErrors
	ch <- numberOfRemoveDropletErrors
}

func (c *HangingDropletsCleaner) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		numberOfRemovedDroplets,
		prometheus.CounterValue,
		float64(c.totalNumberOfRemovedDroplets),
	)

	ch <- prometheus.MustNewConstMetric(
		numberOfStopDropletErrors,
		prometheus.CounterValue,
		float64(c.totalNumberOfStopDropletErrors),
	)

	ch <- prometheus.MustNewConstMetric(
		numberOfRemoveDropletErrors,
		prometheus.CounterValue,
		float64(c.totalNumberOfRemoveDropletErrors),
	)
}

func (c *HangingDropletsCleaner) shouldRemoveDroplet(droplet godo.Droplet, machines []Machine) bool {
	for _, machine := range machines {
		if droplet.Name == machine.Name && machine.DropletId != 0 {
			return false
		}
	}

	return true
}

func (c *HangingDropletsCleaner) stopDroplet(droplet godo.Droplet) {
	logrus.Debugf("Stopping droplet '%s'", droplet.Name)

	if err := c.client.StopDroplet(droplet); err != nil {
		c.totalNumberOfStopDropletErrors++
		logrus.Errorf("Error while stopping droplet '%s': %v", droplet.Name, err.Error())
	}
}

func (c *HangingDropletsCleaner) deleteDroplet(droplet godo.Droplet) {
	logrus.Debugf("Deleting droplet '%s'", droplet.Name)

	if err := c.client.DeleteDroplet(droplet); err != nil {
		c.totalNumberOfRemoveDropletErrors++
		logrus.Errorf("Error while deleting droplet '%s': %v", droplet.Name, err.Error())
		return
	}

	c.totalNumberOfRemovedDroplets++
}

func (c *HangingDropletsCleaner) stopAndDeleteDroplet(droplet godo.Droplet) {
	logrus.Infof("Will stop and delete: %s (created_at: %s)", droplet.Name, droplet.Created)
	if !c.delete {
		return
	}

	c.stopDroplet(droplet)
	c.deleteDroplet(droplet)
}

func (c *HangingDropletsCleaner) findAndDeleteHangingDroplets(droplets []godo.Droplet, machines []Machine) int64 {
	removed := c.totalNumberOfRemovedDroplets
	for _, droplet := range droplets {
		if !c.shouldRemoveDroplet(droplet, machines) {
			continue
		}

		c.stopAndDeleteDroplet(droplet)
	}

	return c.totalNumberOfRemovedDroplets - removed
}

func (c *HangingDropletsCleaner) Clean() error {
	var count int64

	logrus.Infoln("Starting droplets cleanup")
	defer func() {
		logrus.Infof("Finished droplets cleanup. Removed %d droplets", count)
	}()

	machines, err := c.machinesFinder.ListMachines(c.runnerPrefixRegexp)
	if err != nil {
		return err
	}
	logrus.Debugf("Found %d machines matchin prefixes", len(machines))

	droplets, err := c.client.ListDroplets(c.runnerPrefixRegexp, c.dropletAge)
	if err != nil {
		return err
	}
	logrus.Debugf("Found %d droplets matchin prefixes", len(droplets))

	if len(droplets) < 1 {
		return nil
	}

	count = c.findAndDeleteHangingDroplets(droplets, machines)

	return nil
}

func (c *HangingDropletsCleaner) EnableDelete() {
	c.delete = true
}

func NewHangingDropletsCleaner(client client.DigitalOceanClientInterface, machinesFinder MachinesFinderInterface, dropletAge int, runnerPrefix []string) (*HangingDropletsCleaner, error) {
	if len(runnerPrefix) < 1 {
		return nil, fmt.Errorf("You need to set at least one 'runner-prefix'")
	}

	re, err := regexp.Compile(fmt.Sprintf("^(%s)", strings.Join(runnerPrefix, "|")))
	if err != nil {
		return nil, err
	}

	da := time.Duration(dropletAge) * time.Second
	logrus.Infof("Droplet minimal age: %s", da)

	cleaner := &HangingDropletsCleaner{
		client:             client,
		machinesFinder:     machinesFinder,
		runnerPrefix:       runnerPrefix,
		runnerPrefixRegexp: re,
		dropletAge:         da,
	}

	return cleaner, err
}
