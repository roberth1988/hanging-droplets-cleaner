package commands

import (
	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/tmaczukin/hanging-droplets-cleaner/cleaner"
	"gitlab.com/tmaczukin/hanging-droplets-cleaner/client"
)

type CleanerProvider struct{}

func (s *CleanerProvider) GetCleaner(context *cli.Context) *cleaner.HangingDropletsCleaner {
	apiToken := context.String("digitalocean-token")
	if apiToken == "" {
		logrus.Fatalln("Missing DigitalOcean API Token. Exiting")
	}

	var err error
	cleaner, err := cleaner.NewHangingDropletsCleaner(
		client.NewDigitalOceanClient(apiToken),
		cleaner.NewMachinesFinder(context.String("machines-directory")),
		context.Int("droplet-age"),
		context.StringSlice("runner-prefix"),
	)

	if err != nil {
		logrus.Fatalf("Failed to start HangingDropletsCleaner: %v", err.Error())
	}

	return cleaner
}

func (s *CleanerProvider) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:   "digitalocean-token",
			Usage:  "DigitalOcean API Token",
			EnvVars: []string{
				"DIGITALOCEAN_TOKEN",
			},
		},
		&cli.StringFlag{
			Name:   "machines-directory",
			Usage:  "Absolute path to directory where Docker Machine machines configuration is stored",
			Value:  "/root/.docker/machine/machines",
			EnvVars: []string{
				"MACHINES_DIRECTORY",
			},
		},
		&cli.IntFlag{
			Name:   "droplet-age",
			Usage:  "Minimal age of droplet that can be removed",
			Value:  DefaultInterval,
			EnvVars: []string{
				"DROPLET_AGE",
			},
		},
		&cli.StringSliceFlag{
			Name:  "runner-prefix",
			Usage: "Prefix of runner's droplet name",
		},
	}
}
