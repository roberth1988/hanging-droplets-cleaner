package cleaner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

type MachinesFinderInterface interface {
	ListMachines(*regexp.Regexp) ([]Machine, error)
	GetMachinesDirectory() string
}

type MachinesFinder struct {
	machinesDirectory string
}

type Machine struct {
	Name      string
	DropletId float64
}

func (m *MachinesFinder) ListMachines(runnerPrefixRegexp *regexp.Regexp) ([]Machine, error) {
	entries, err := ioutil.ReadDir(m.machinesDirectory)
	if err != nil {
		return nil, err
	}

	var machines []Machine

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || !runnerPrefixRegexp.MatchString(name) {
			continue
		}

		configFile := fmt.Sprintf("%s/%s/config.json", m.machinesDirectory, name)
		dropletId := float64(0)

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return nil, err
		} else {
			dockerMachineConfigFile, err := os.Open(configFile)
			if err != nil {
				return nil, err
			}

			jsonByteValue, _ := ioutil.ReadAll(dockerMachineConfigFile)
			dockerMachineConfigFile.Close()

			var dockerMachineConfigParsed map[string]interface{}
			err = json.Unmarshal([]byte(jsonByteValue), &dockerMachineConfigParsed)

			if err != nil {
				return nil, err
			}

			if driverConfig, ok := dockerMachineConfigParsed["Driver"].(map[string]interface{}); ok {

				dropletIdString := driverConfig["DropletID"].(float64)

				if dropletIdString != 0 {
					dropletId = dropletIdString
				}
			}

		}

		machines = append(machines, Machine{
			Name:      name,
			DropletId: dropletId,
		})
	}

	return machines, nil
}

func (m *MachinesFinder) GetMachinesDirectory() string {
	return m.machinesDirectory
}

func NewMachinesFinder(machinesDirectory string) *MachinesFinder {
	return &MachinesFinder{
		machinesDirectory: machinesDirectory,
	}
}
