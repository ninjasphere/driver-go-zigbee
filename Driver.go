package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-zigbee"
	"github.com/ninjasphere/go-zigbee/gateway"
	"github.com/ninjasphere/go-zigbee/nwkmgr"
)

const (
	ClusterIDBasic    uint32 = 0x00
	ClusterIDOnOff    uint32 = 0x06
	ClusterIDTemp     uint32 = 0x402
	ClusterIDHumidity uint32 = 0x405
	ClusterIDPower    uint32 = 0x702
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

	NetworkReady chan bool

	nwkmgrConn  *zigbee.ZStackNwkMgr
	gatewayConn *zigbee.ZStackGateway
	otaConn     *zigbee.ZStackOta
}

func NewDriver() *Driver {
	return &Driver{
		NetworkReady: make(chan bool),
	}
}

type Device struct {
	ManufacturerName string
	ModelIdentifier  string

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

func (d *Driver) Connect(cfg *ZStackConfig, networkReady chan bool) error {

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
	d.nwkmgrConn.OnDeviceFound = func(deviceInfo *nwkmgr.NwkDeviceInfoT) {
		d.onDeviceFound(deviceInfo)
	}

	done := false
	d.nwkmgrConn.OnNetworkReady = func() {
		if !done {
			done = true
			networkReady <- true
		}
	}

	d.otaConn, err = zigbee.ConnectToOtaServer(cfg.Hostname, cfg.OtasrvrPort)
	if err != nil {
		fmt.Errorf("Error connecting to ota server %s", err)
	}

	d.gatewayConn, err = zigbee.ConnectToGatewayServer(cfg.Hostname, cfg.GatewayPort)
	if err != nil {
		log.Printf("Error connecting to gateway %s", err)
	}

	return nil

}

func (d *Driver) FetchDevices() error {

	return d.nwkmgrConn.FetchDeviceList()

}

func (d *Device) getBasicInfo() error {

	log.Printf("Getting basic information from %X", *d.deviceInfo.IeeeAddress)

	cluster := ClusterIDBasic
	ManufacturerNameAttribute := uint32(0x004)
	ModelIdentifierAttribute := uint32(0x005)

	request := &gateway.GwReadDeviceAttributeReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    d.deviceInfo.IeeeAddress,
		},
		ClusterId:     &cluster,
		AttributeList: []uint32{ManufacturerNameAttribute, ModelIdentifierAttribute},
	}

	response := &gateway.GwReadDeviceAttributeRspInd{}
	err := d.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting basic device information state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get basic device information. status: %s", response.Status.String())
	}

	for _, attribute := range response.AttributeRecordList {

		switch *attribute.AttributeId {
		case ManufacturerNameAttribute:
			d.ManufacturerName = string(attribute.AttributeValue)
		case ModelIdentifierAttribute:
			d.ModelIdentifier = string(attribute.AttributeValue)
		default:
			log.Printf("Unknown attribute returned when finding basic info %s", *attribute.AttributeId)
		}
	}

	return nil
}

func (d *Driver) onDeviceFound(deviceInfo *nwkmgr.NwkDeviceInfoT) {

	fmt.Printf("\n\n")
	fmt.Printf("---- Found Device IEEE:%X ----\f", *deviceInfo.IeeeAddress)

	/*sigs, _ := simplejson.NewJson([]byte(`{
	    "ninja:manufacturer": "Phillips",
	    "ninja:productName": "Hue",
	    "manufacturer:productModelId": "",
	    "ninja:productType": "Light",
	    "ninja:thingType": "light"
	}`))*/

	device := &Device{
		driver:     d,
		deviceInfo: deviceInfo,
	}

	err := device.getBasicInfo()
	if err != nil {
		log.Printf("Failed to get basic info: %s", err)
	}

	sigs, _ := simplejson.NewJson([]byte(`{
	}`))
	if device.ManufacturerName != "" {
		sigs.Set("zigbee:ManufacturerName", device.ManufacturerName)
	} else {
		device.ManufacturerName = "Unknown"
	}
	if device.ModelIdentifier != "" {
		sigs.Set("zigbee:ModelIdentifier", device.ModelIdentifier)
	} else {
		device.ModelIdentifier = fmt.Sprintf("MAC:%X", *deviceInfo.IeeeAddress)
	}

	name := fmt.Sprintf("%s by %s", device.ModelIdentifier, device.ManufacturerName)

	spew.Dump(deviceInfo)

	device.bus, _ = d.bus.AnnounceDevice(fmt.Sprintf("%X", *deviceInfo.IeeeAddress), "zigbee", name, sigs)

	log.Printf("Got device : %d", *deviceInfo.IeeeAddress)

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

		if containsUInt32(endpoint.InputClusters, ClusterIDTemp) {
			log.Printf("This endpoint has temperature cluster")

			temp := &TempChannel{
				Channel{
					device:   device,
					endpoint: endpoint,
				},
			}

			err := temp.init()
			if err != nil {
				log.Printf("Failed initialising temp channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDHumidity) {
			log.Printf("This endpoint has humidity cluster")

			temp := &TempChannel{
				Channel{
					device:   device,
					endpoint: endpoint,
				},
			}

			err := temp.init()
			if err != nil {
				log.Printf("Failed initialising humidity channel: %s", err)
			}

		}

	}

	d.devices = append(d.devices, device)

	fmt.Printf("---- Finished Device IEEE:%X ----\n", *deviceInfo.IeeeAddress)

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
