package main

import (
	"github.com/ninjasphere/go-ninja/api"
	nconfig "github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/support"
	"os"
)

var (
	info   = ninja.LoadModuleInfo("./package.json")
	log    = logger.GetLogger(info.ID) // gets replaced by NewDriver
	config = &ZStackConfig{
		Hostname:       "localhost",
		OtasrvrPort:    2525,
		GatewayPort:    2541,
		NwkmgrPort:     2540,
		StableFlagFile: "/var/run/zigbee.stable", // TODO
	}
)

func main() {
	config.StableFlagFile = nconfig.String("/var/run/zigbee.stable", "zigbee", "stable-file")
	config.Hostname = nconfig.String("localhost", "zigbee", "host")

	check, err := os.Open("/etc/disable-zigbee")
	if err != nil {

		// this is the expected case

		_, err := NewDriver(config)
		if log.IsDebugEnabled() {
			log.Debugf("version - %s - running with configuration %+v", Version, config)
		}

		if err != nil {
			log.Fatalf("Failed to start ZigBee driver: %s", err)
		}
	} else {
		check.Close()
		log.Debugf("version - %s - zigbee access disabled by /etc/disable-zigbee", Version)
	}

	support.WaitUntilSignal()
}
