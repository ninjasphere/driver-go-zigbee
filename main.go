package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/davecgh/go-spew/spew"
)

var _ = fmt.Printf
var _ = spew.Dump

var config = &ZStackConfig{
	Hostname:    "elliotdev.local",
	OtasrvrPort: 2525,
	GatewayPort: 2541,
	NwkmgrPort:  2540,
}

func main() {

	//spew.Dump(bus)

	driver := NewDriver()

	networkReady := make(chan bool)

	err := driver.Connect(config, networkReady)
	if err != nil {
		log.Fatalf("Failed to start ZigBee driver: %s", err)
	}

	//driver.Reset(true)

	//driver.FetchDevices()

	/*err = driver.PermitJoin(120)
	if err != nil {
		log.Fatalf("Failed to enable joining: %s", err)
	}*/

	//driver.Reset(false)

	//log.Println("Waiting!")

	//select {
	//case <-networkReady:
	log.Println("Driver starting!")
	err = driver.PermitJoin(120)
	if err != nil {
		log.Fatalf("Failed to enable joining: %s", err)
	}
	driver.FetchDevices()
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
