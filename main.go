package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/ninjasphere/go-ninja/api"
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
	flagset := flag.NewFlagSet("driver-go-zigbee", flag.ContinueOnError)
	flagset.StringVar(&config.StableFlagFile, "zigbee-stable-file", "/var/run/zigbee.stable", "Location of zigbee.stable")
	flagset.StringVar(&config.Hostname, "zstack-host", "localhost", "IP address or DNS name of zstack host.")
	flagset.Parse(os.Args[1:])

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
