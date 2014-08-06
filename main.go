package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja"
)

var _ = fmt.Printf
var _ = spew.Dump

func main() {

	conn, err := ninja.Connect("com.ninjablocks.zigbee")
	if err != nil {
		log.Fatalf("Could not connect to MQTT: %s", err)
	}

	bus, err := conn.AnnounceDriver("com.ninjablocks.zigbee", "driver-zigbee", getCurDir())
	if err != nil {
		log.Fatalf("Could not get driver bus: %s", err)
	}

	spew.Dump(bus)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)

}

func getCurDir() string {
	pwd, _ := os.Getwd()
	return pwd + "/"
}
