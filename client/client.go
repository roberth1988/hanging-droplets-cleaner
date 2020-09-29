package client

import (
	"context"
	"regexp"
	"time"

	"github.com/digitalocean/godo"

	"golang.org/x/oauth2"

	"gitlab.com/tmaczukin/hanging-droplets-cleaner/version"
)

type tokenSource struct {
	accessToken string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.accessToken,
	}, nil
}

type DigitalOceanClientInterface interface {
	ListDroplets(*regexp.Regexp, time.Duration) ([]godo.Droplet, error)
	StopDroplet(godo.Droplet) error
	DeleteDroplet(godo.Droplet) error
}

type DigitalOceanClient struct {
	client *godo.Client
}

func (c *DigitalOceanClient) selectDroplets(dropletsPrefixRegexp *regexp.Regexp, dropletAge time.Duration, dropletsList []godo.Droplet) (droplets []godo.Droplet) {
	for _, droplet := range dropletsList {
		if !dropletsPrefixRegexp.MatchString(droplet.Name) {
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, droplet.Created)
		if err != nil || time.Now().Sub(createdAt) < dropletAge {
			continue
		}

		droplets = append(droplets, droplet)
	}

	return droplets
}

func (c *DigitalOceanClient) listDropletsPage(dropletsPrefixRegexp *regexp.Regexp, dropletAge time.Duration, pageOpts *godo.ListOptions) (droplets []godo.Droplet, readNext bool, err error) {
	readNext = false
	dropletsList, resp, err := c.client.Droplets.List(context.TODO(), pageOpts)
	if err != nil {
		return
	}

	droplets = c.selectDroplets(dropletsPrefixRegexp, dropletAge, dropletsList)

	if resp.Links == nil || resp.Links.IsLastPage() {
		return
	}

	page, err := resp.Links.CurrentPage()
	if err != nil {
		return
	}

	pageOpts.Page = page + 1
	readNext = true

	return
}

func (c *DigitalOceanClient) ListDroplets(dropletsPrefixRegexp *regexp.Regexp, dropletAge time.Duration) (droplets []godo.Droplet, err error) {
	pageOpts := &godo.ListOptions{
		Page:    1,
		PerPage: 250,
	}

	var selectedDroplets []godo.Droplet
	var readNext bool
	for {
		selectedDroplets, readNext, err = c.listDropletsPage(dropletsPrefixRegexp, dropletAge, pageOpts)
		if err != nil {
			return
		}

		droplets = append(droplets, selectedDroplets...)

		if !readNext {
			break
		}
	}

	return
}

func (c *DigitalOceanClient) StopDroplet(droplet godo.Droplet) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancelFn()

	_, _, err := c.client.DropletActions.PowerOff(ctx, droplet.ID)

	return err
}

func (c *DigitalOceanClient) DeleteDroplet(droplet godo.Droplet) error {
	_, err := c.client.Droplets.Delete(context.TODO(), droplet.ID)
	return err
}

func NewDigitalOceanClient(apiToken string) *DigitalOceanClient {
	ts := &tokenSource{accessToken: apiToken}
	client := godo.NewClient(oauth2.NewClient(oauth2.NoContext, ts))
	client.UserAgent = version.AppVersion.UserAgent()

	return &DigitalOceanClient{
		client: client,
	}
}
