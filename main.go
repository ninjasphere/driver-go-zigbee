package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/ninjasphere/go-ninja/api"
	nconfig "github.com/ninjasphere/go-ninja/config"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-zigbee"
)

var (
	info   = ninja.LoadModuleInfo("./package.json")
	log    = logger.GetLogger(info.ID)
	config = &ZStackConfig{
		Hostname:       "localhost",
		OtasrvrPort:    2525,
		GatewayPort:    2541,
		NwkmgrPort:     2540,
		StableFlagFile: "/var/run/zigbee.stable", // TODO
	}
)

func main() {
	// do all zigbee logger with the driver's logger

	zigbee.SetLogger(logger.GetLogger(info.ID + ".backend"))

	// FIXME: use ninja configuration framework
	config.StableFlagFile = nconfig.String("/var/run/zigbee.stable", "zigbee", "stable-file")
	config.Hostname = nconfig.String("localhost", "zigbee", "host")

	_, err := NewDriver(config)
	if err != nil {
		log.Fatalf("Failed to start ZigBee driver: %s", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)
}
