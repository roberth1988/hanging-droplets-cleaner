package cleaner

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

type FakeDOClient struct {
	t                    *testing.T
	listDropletsAsserts  func(*FakeDOClient) ([]godo.Droplet, error)
	stopDropletAsserts   func(*FakeDOClient, godo.Droplet) error
	deleteDropletAsserts func(*FakeDOClient, godo.Droplet) error
}

func (fc *FakeDOClient) ListDroplets(dropletsPrefixRegexp *regexp.Regexp, dropletAge time.Duration) ([]godo.Droplet, error) {
	if fc.listDropletsAsserts != nil {
		return fc.listDropletsAsserts(fc)
	}
	return []godo.Droplet{}, nil
}

func (fc *FakeDOClient) StopDroplet(droplet godo.Droplet) error {
	if fc.stopDropletAsserts != nil {
		return fc.stopDropletAsserts(fc, droplet)
	}
	return nil
}

func (fc *FakeDOClient) DeleteDroplet(droplet godo.Droplet) error {
	if fc.deleteDropletAsserts != nil {
		return fc.deleteDropletAsserts(fc, droplet)
	}
	return nil
}

type FakeMachinesFinder struct {
	t                   *testing.T
	listMachinesAsserts func(*FakeMachinesFinder) ([]Machine, error)
}

func (m *FakeMachinesFinder) ListMachines(runnerPrefixRegexp *regexp.Regexp) ([]Machine, error) {
	if m.listMachinesAsserts != nil {
		return m.listMachinesAsserts(m)
	}
	return []Machine{}, nil
}

func getCleaner(t *testing.T) (cleaner *HangingDropletsCleaner, client *FakeDOClient, machinesFinder *FakeMachinesFinder) {
	client = &FakeDOClient{t: t}
	machinesFinder = &FakeMachinesFinder{t: t}

	var err error
	cleaner, err = NewHangingDropletsCleaner(
		client,
		machinesFinder,
		10,
		[]string{"runner-abc123"},
	)
	assert.NoError(t, err)

	return
}

func TestCleanerNoMachines(t *testing.T) {
	cleaner, client, _ := getCleaner(t)
	cleaner.EnableDelete()

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-1", Created: time.Now().Format(time.RFC3339)},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.True(t, deleteDropletCalled, "DeleteDroplet() should be called")
	assert.True(t, stopDropletCalled, "StopDroplet() should be called")
}

func TestCleanerNoMachinesAndDeleteDisabled(t *testing.T) {
	cleaner, client, _ := getCleaner(t)

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-1", Created: time.Now().Format(time.RFC3339)},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.False(t, deleteDropletCalled, "DeleteDroplet() should not be called")
	assert.False(t, stopDropletCalled, "StopDroplet() should not be called")
}

func TestCleanerNoDroplets(t *testing.T) {
	cleaner, client, machinesFinder := getCleaner(t)
	cleaner.EnableDelete()

	machinesFinder.listMachinesAsserts = func(*FakeMachinesFinder) (machines []Machine, err error) {
		machines = []Machine{
			{
				Name:      "runner-abc123-test-1",
				DropletId: "",
			},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.False(t, deleteDropletCalled, "DeleteDroplet() should not be called")
	assert.False(t, stopDropletCalled, "StopDroplet() should not be called")
}

func TestCleanerDropletsAndMachinesNotMatching(t *testing.T) {
	cleaner, client, machinesFinder := getCleaner(t)
	cleaner.EnableDelete()

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-2", Created: time.Now().Format(time.RFC3339)},
			{ID: 2, Name: "runner-abc123-test-3", Created: time.Now().Format(time.RFC3339)},
		}
		return
	}

	machinesFinder.listMachinesAsserts = func(*FakeMachinesFinder) (machines []Machine, err error) {
		machines = []Machine{
			{
				Name:      "runner-abc123-test-1",
				DropletId: "0",
			},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.True(t, stopDropletCalled, "StopDroplet() should be called")
	assert.True(t, deleteDropletCalled, "DeleteDroplet() should be called")
	assert.Equal(t, int64(2), cleaner.totalNumberOfRemovedDroplets, "Should remove all droplets missing machine")
}

func TestCleanerDropletsAndMachinesPartiallyMatching(t *testing.T) {
	cleaner, client, machinesFinder := getCleaner(t)
	cleaner.EnableDelete()

	dropletToBeRemoved := godo.Droplet{ID: 2, Name: "runner-abc123-test-2", Created: time.Now().Format(time.RFC3339)}

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-1", Created: time.Now().Format(time.RFC3339)},
			dropletToBeRemoved,
		}
		return
	}

	machinesFinder.listMachinesAsserts = func(*FakeMachinesFinder) (machines []Machine, err error) {
		machines = []Machine{
			{
				Name:      "runner-abc123-test-1",
				DropletId: "0",
			},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		assert.Equal(t, droplet, dropletToBeRemoved, "Should stop only specified droplet")
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		assert.Equal(t, droplet, dropletToBeRemoved, "Should remove only specified droplet")
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.True(t, stopDropletCalled, "StopDroplet() should be called")
	assert.True(t, deleteDropletCalled, "DeleteDroplet() should be called")
	assert.Equal(t, int64(1), cleaner.totalNumberOfRemovedDroplets, "Should remove only matching droplets")
}

func TestCleanerDropletsAndMachinesMatching(t *testing.T) {
	cleaner, client, machinesFinder := getCleaner(t)
	cleaner.EnableDelete()

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-1", Created: time.Now().Format(time.RFC3339)},
			{ID: 2, Name: "runner-abc123-test-2", Created: time.Now().Format(time.RFC3339)},
		}
		return
	}

	machinesFinder.listMachinesAsserts = func(*FakeMachinesFinder) (machines []Machine, err error) {
		machines = []Machine{
			{
				Name:      "runner-abc123-test-1",
				DropletId: "0",
			},
			{
				Name:      "runner-abc123-test-2",
				DropletId: "0",
			},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.False(t, stopDropletCalled, "StopDroplet() should not be called")
	assert.False(t, deleteDropletCalled, "DeleteDroplet() should not be called")
}

func TestErrorOnMachineStop(t *testing.T) {
	cleaner, client, _ := getCleaner(t)
	cleaner.EnableDelete()

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-1", Created: time.Now().Format(time.RFC3339)},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return errors.New("error on machine stop")
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.True(t, deleteDropletCalled, "DeleteDroplet() should be called")
	assert.True(t, stopDropletCalled, "StopDroplet() should be called")
	assert.Equal(t, int64(1), cleaner.totalNumberOfStopDropletErrors, "Should count stop errors")
	assert.Equal(t, int64(0), cleaner.totalNumberOfRemoveDropletErrors, "There should be no delete errors")
	assert.Equal(t, int64(1), cleaner.totalNumberOfRemovedDroplets, "Should remove all droplets")
}

func TestErrorOnMachineDelete(t *testing.T) {
	cleaner, client, _ := getCleaner(t)
	cleaner.EnableDelete()

	client.listDropletsAsserts = func(c *FakeDOClient) (droplets []godo.Droplet, err error) {
		droplets = []godo.Droplet{
			{ID: 1, Name: "runner-abc123-test-1", Created: time.Now().Format(time.RFC3339)},
		}
		return
	}

	stopDropletCalled := false
	client.stopDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		stopDropletCalled = true
		return
	}

	deleteDropletCalled := false
	client.deleteDropletAsserts = func(c *FakeDOClient, droplet godo.Droplet) (err error) {
		deleteDropletCalled = true
		err = errors.New("error on machine delete")
		return
	}

	err := cleaner.Clean()
	assert.NoError(t, err)
	assert.True(t, deleteDropletCalled, "DeleteDroplet() should be called")
	assert.True(t, stopDropletCalled, "StopDroplet() should be called")
	assert.Equal(t, int64(0), cleaner.totalNumberOfStopDropletErrors, "There should be no stop errors")
	assert.Equal(t, int64(1), cleaner.totalNumberOfRemoveDropletErrors, "Should count delete errors")
	assert.Equal(t, int64(0), cleaner.totalNumberOfRemovedDroplets, "There should be no deletes")
}
