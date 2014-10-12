package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/davecgh/go-spew/spew"
)

var _ = fmt.Printf
var _ = spew.Dump

var config = &ZStackConfig{
	Hostname:       "localhost",
	OtasrvrPort:    2525,
	GatewayPort:    2541,
	NwkmgrPort:     2540,
	StableFlagFile: "/var/run/zigbee.stable",
}

func main() {

	flagset := flag.NewFlagSet("drive-go-zigbee", flag.ContinueOnError)
	flagset.StringVar(&config.StableFlagFile, "zigbee-stable-file", "/var/run/zigbee.stable", "Location of zigbee.stable")
	flagset.StringVar(&config.Hostname, "zstack-host", "localhost", "IP address or DNS name of zstack host.")
	flagset.Parse(os.Args)

	//spew.Dump(bus)

	driver := NewDriver()

	networkReady := make(chan bool)

	err := driver.Connect(config, networkReady)
	if err != nil {
		log.Fatalf("Failed to start ZigBee driver: %s", err)
	}

	//	driver.Reset(true)

	//driver.FetchDevices()

	/*err = driver.PermitJoin(120)
	if err != nil {
		log.Fatalf("Failed to enable joining: %s", err)
	}*/

	//	driver.Reset(true)

	//log.Println("Waiting!")

	//select {
	//case <-networkReady:
	log.Println("Driver starting!")
	driver.FetchDevices()
	err = driver.PermitJoin(120)
	if err != nil {
		log.Fatalf("Failed to enable joining: %s", err)
	}
	//case <-time.After(20 * time.Second):
	//log.Println("Timeout waiting for network ready. Soft-resetting cc2530")
	//driver.Reset(false)
	//os.Exit(0)
	//}

	/*

				toggleRequest := &gateway.DevSetOnOffStateReq{
					DstAddress: &gateway.GwAddressStructT{
						AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
						IeeeAddr:    device.IeeeAddress,
					},
					State: gateway.GwOnOffStateT_TOGGLE_STATE.Enum(),
				}

				confirmation := &gateway.GwZigbeeGenericCnf{}

				err = gatewayConn.SendCommand(toggleRequest, confirmation)
				if err != nil {
					log.Fatalf("Failed to toggle device: ", err)
				}
				log.Printf("Got on/off confirmation")
				if confirmation.Status.String() != "STATUS_SUCCESS" {
					log.Fatalf("Failed to request the device to toggle. Status:%s", confirmation.Status.String())
				}

				response := &gateway.GwZigbeeGenericRspInd{}
				err = gatewayConn.WaitForSequenceResponse(confirmation.SequenceNumber, response)
				if err != nil {
					log.Fatalf("Failed to get on/off response: ", err)
				}

				log.Printf("Got toggle response from device! Status: %s", response.Status.String())

				spew.Dump(response)

				powerRequest := &gateway.DevGetPowerReq{
					DstAddress: &gateway.GwAddressStructT{
						AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
						IeeeAddr:    device.IeeeAddress,
					},
				}

				confirmation = &gateway.GwZigbeeGenericCnf{}

				err = gatewayConn.SendCommand(powerRequest, confirmation)
				if err != nil {
					log.Fatalf("Failed to request power: ", err)
				}
				log.Printf("Got power request confirmation")
				if confirmation.Status.String() != "STATUS_SUCCESS" {
					log.Fatalf("Failed to request the power. Status:%s", confirmation.Status.String())
				}

				powerResponse := &gateway.DevGetPowerRspInd{}
				err = gatewayConn.WaitForSequenceResponse(confirmation.SequenceNumber, powerResponse)
				if err != nil {
					log.Fatalf("Failed to get power response: ", err)
				}

				spew.Dump(powerResponse)

	}*/

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)

}
