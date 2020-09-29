package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

type OneShotCommand struct {
	provider *CleanerProvider
}

func (o *OneShotCommand) Confirm(message string) bool {
	fmt.Print(fmt.Sprintf("%s [yes/no] -> ", message))

	reader := bufio.NewReader(os.Stdin)
	data, _, err := reader.ReadLine()
	if err != nil {
		logrus.Fatalf("Error on reading user input: %v", err.Error())
	}

	result := strings.ToLower(strings.TrimSpace(string(data)))

	return result == "yes"
}

func (o *OneShotCommand) Execute(context *cli.Context) {
	logrus.Infoln("Running in one-shot mode")

	cleaner := o.provider.GetCleaner(context)

	if context.Bool("delete") && o.Confirm("Are you sure you want to delete droplets?") {
		logrus.Warnln("Running with 'delete' flag. All droplets matching requirements will be removed!")
		cleaner.EnableDelete()
	} else {
		logrus.Infoln("Running without 'delete' flag. Will not remove any droplet.")
	}

	if err := cleaner.Clean(); err != nil {
		logrus.Fatalf("Error during cleanup: %v", err.Error())
	}
}

func NewOneShotCommand() cli.Command {
	provider := &CleanerProvider{}
	cmd := &OneShotCommand{
		provider: provider,
	}

	flags := []cli.Flag{
		&cli.BoolFlag{
			Name:  "delete",
			Usage: "Delete droplets",
		},
	}
	flags = append(flags, provider.Flags()...)

	return cli.Command{
		Name:   "one-shot",
		Usage:  "Start hanging droplets cleaner in a one-shot mode",
		Action: func(c *cli.Context) error {
			cmd.Execute(c)
			return nil
		},
		Flags:  flags,
	}
}
