package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-zigbee"
	"github.com/ninjasphere/go-zigbee/gateway"
	"github.com/ninjasphere/go-zigbee/nwkmgr"
)

var _ = fmt.Printf
var _ = spew.Dump

const (
	hostname    = "beaglebone.local"
	otasrvrPort = 2525
	gatewayPort = 2541
	nwkmgrPort  = 2540
)

func main() {

	conn, err := ninja.Connect("com.ninjablocks.zigbee")
	if err != nil {
		log.Fatalf("Could not connect to MQTT: %s", err)
	}

	_, err = conn.AnnounceDriver("com.ninjablocks.zigbee", "driver-zigbee", getCurDir())
	if err != nil {
		log.Fatalf("Could not get driver bus: %s", err)
	}

	//spew.Dump(bus)

	nwkmgrConn, err := zigbee.ConnectToNwkMgrServer(hostname, nwkmgrPort)
	if err != nil {
		// handle error
		log.Printf("Error connecting to nwkmgr %s", err)
	}

	gatewayConn, err := zigbee.ConnectToGatewayServer(hostname, gatewayPort)
	if err != nil {
		// handle error
		log.Printf("Error connecting to gateway %s", err)
	}

	deviceListResponse := &nwkmgr.NwkGetDeviceListCnf{}

	err = nwkmgrConn.SendCommand(&nwkmgr.NwkGetDeviceListReq{}, deviceListResponse)
	if err != nil {
		log.Fatalf("Failed to get device list: %s", err)
	}
	log.Printf("Found %d device(s): ", len(deviceListResponse.DeviceList))
	spew.Dump(deviceListResponse)

	for _, device := range deviceListResponse.DeviceList {
		log.Printf("Got device : %d", device.IeeeAddress)
		for _, endpoint := range device.SimpleDescList {
			log.Printf("Got endpoint : %d", endpoint.EndpointId)

			if containsUInt32(endpoint.InputClusters, 0x06) {
				log.Printf("This endpoint has on/off cluster")

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
			}

		}
	}

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

func containsUInt32(hackstack []uint32, needle uint32) bool {
	for _, cluster := range hackstack {
		if cluster == needle {
			return true
		}
	}
	return false
}
