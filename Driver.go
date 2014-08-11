package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bitly/go-simplejson"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-zigbee"
	"github.com/ninjasphere/go-zigbee/nwkmgr"
)

const (
	ClusterIDOnOff uint32 = 0x06
	ClusterIDPower uint32 = 0x702
)

type ZStackConfig struct {
	Hostname    string
	OtasrvrPort int
	GatewayPort int
	NwkmgrPort  int
}

type Driver struct {
	conn *ninja.NinjaConnection
	bus  *ninja.DriverBus

	devices []*Device

	nwkmgrConn  *zigbee.ZStackNwkMgr
	gatewayConn *zigbee.ZStackGateway
	otaConn     *zigbee.ZStackOta
}

type Device struct {
	driver     *Driver
	deviceInfo *nwkmgr.NwkDeviceInfoT
	bus        *ninja.DeviceBus
	channels   []Channel
}

type Channel struct {
	device   *Device
	endpoint *nwkmgr.NwkSimpleDescriptorT
	bus      *ninja.ChannelBus
}

func (d *Driver) Reset(hard bool) error {
	return d.nwkmgrConn.Reset(hard)
}

func (d *Driver) PermitJoin(time uint32) error {

	log.Printf("Pemitting join for %d seconds", time)

	//joinTime := uint32(30)
	permitJoinRequest := &nwkmgr.NwkSetPermitJoinReq{
		PermitJoinTime: &time,
		PermitJoin:     nwkmgr.NwkPermitJoinTypeT_PERMIT_ALL.Enum(),
	}

	permitJoinResponse := &nwkmgr.NwkZigbeeGenericCnf{}

	err := d.nwkmgrConn.SendCommand(permitJoinRequest, permitJoinResponse)
	if err != nil {
		return fmt.Errorf("Failed to enable joining: %s", err)
	}
	if permitJoinResponse.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to enable joining: %s", permitJoinResponse.Status)
	}
	//log.Println("Permit join response: ")
	//outJSON(permitJoinResponse)

	return nil
}

func (d *Driver) Connect(cfg *ZStackConfig) error {

	conn, err := ninja.Connect("com.ninjablocks.zigbee")
	if err != nil {
		return fmt.Errorf("Could not connect to MQTT: %s", err)
	}

	d.conn = conn

	d.bus, err = conn.AnnounceDriver("com.ninjablocks.zigbee", "driver-zigbee", getCurDir())
	if err != nil {
		return fmt.Errorf("Could not get driver bus: %s", err)
	}

	d.nwkmgrConn, err = zigbee.ConnectToNwkMgrServer(cfg.Hostname, cfg.NwkmgrPort)
	if err != nil {
		return fmt.Errorf("Error connecting to nwkmgr %s", err)
	}
	d.nwkmgrConn.OnDeviceFound = d.onDeviceFound

	d.otaConn, err = zigbee.ConnectToOtaServer(cfg.Hostname, cfg.OtasrvrPort)
	if err != nil {
		fmt.Errorf("Error connecting to ota server %s", err)
	}

	d.gatewayConn, err = zigbee.ConnectToGatewayServer(cfg.Hostname, cfg.GatewayPort)
	if err != nil {
		log.Printf("Error connecting to gateway %s", err)
	}

	d.nwkmgrConn.FetchDeviceList()

	return nil

}

func (d *Driver) onDeviceFound(deviceInfo *nwkmgr.NwkDeviceInfoT) {

	/*sigs, _ := simplejson.NewJson([]byte(`{
	    "ninja:manufacturer": "Phillips",
	    "ninja:productName": "Hue",
	    "manufacturer:productModelId": "",
	    "ninja:productType": "Light",
	    "ninja:thingType": "light"
	}`))*/

	sigs, _ := simplejson.NewJson([]byte(`{
  }`))

	deviceBus, _ := d.bus.AnnounceDevice(fmt.Sprintf("%X", *deviceInfo.IeeeAddress), "zigbee", "Unknown Device 2 FIXME", sigs)

	device := &Device{
		driver:     d,
		deviceInfo: deviceInfo,
		bus:        deviceBus,
	}

	log.Printf("Got device : %d", *deviceInfo.IeeeAddress)

	spew.Dump(deviceInfo)

	for _, endpoint := range deviceInfo.SimpleDescList {
		log.Printf("Got endpoint : %d", *endpoint.EndpointId)

		if containsUInt32(endpoint.InputClusters, ClusterIDOnOff) {
			log.Printf("This endpoint has on/off cluster")

			onOff := &OnOffChannel{
				Channel{
					device:   device,
					endpoint: endpoint,
				},
			}

			err := onOff.init()
			if err != nil {
				log.Printf("Failed initialising on/off channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDPower) {
			log.Printf("This endpoint has power cluster")

			power := &PowerChannel{
				Channel{
					device:   device,
					endpoint: endpoint,
				},
			}

			err := power.init()
			if err != nil {
				log.Printf("Failed initialising power channel: %s", err)
			}

		}

	}

	d.devices = append(d.devices, device)

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
