package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
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

var (
	info = ninja.LoadModuleInfo("./package.json")
)

type ZStackConfig struct {
	Hostname       string
	OtasrvrPort    int
	GatewayPort    int
	NwkmgrPort     int
	StableFlagFile string
}

type DriverConfig struct {
}

type Driver struct {
	conn      *ninja.Connection
	sendEvent func(event string, payload interface{}) error

	devices map[uint64]*Device

	NetworkReady chan bool

	nwkmgrConn  *zigbee.ZStackNwkMgr
	gatewayConn *zigbee.ZStackGateway
	otaConn     *zigbee.ZStackOta
	config      *DriverConfig
}

func NewDriver() *Driver {
	return &Driver{
		NetworkReady: make(chan bool),
	}
}

type Device struct {
	info       *model.Device
	driver     *Driver
	deviceInfo *nwkmgr.NwkDeviceInfoT
	channels   []Channel
	sendEvent  func(event string, payload interface{}) error
}

type Channel struct {
	device   *Device
	endpoint *nwkmgr.NwkSimpleDescriptorT
}

func (d *Driver) GetModuleInfo() *model.Module {
	return info
}

func (d *Driver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}

func (d *Driver) Reset(hard bool) error {
	return d.nwkmgrConn.Reset(hard)
}

func (d *Driver) Start(config *DriverConfig) error {
	d.config = config
	return d.sendEvent("config", config)
}

func (d *Driver) Stop() error {
	return fmt.Errorf("This driver does not support being stopped. YOU HAVE NO POWER HERE.")
}

func (d *Driver) StartPairing(period uint32) (*uint32, error) {
	err := d.PermitJoin(uint32(period))
	if err != nil {
		err = fmt.Errorf("permit Join failed: %v", err)
		log.Fatalf("%s", err)
	}
	return &period, err
}

func (d *Driver) PermitJoin(period uint32) error {

	log.Printf("Join window will open for %d seconds.", period)

	//joinTime := uint32(30)
	permitJoinRequest := &nwkmgr.NwkSetPermitJoinReq{
		PermitJoinTime: &period,
		PermitJoin:     nwkmgr.NwkPermitJoinTypeT_PERMIT_ALL.Enum(),
	}

	permitJoinResponse := &nwkmgr.NwkZigbeeGenericCnf{}

	go func() {
		// logging the close of the join window is helpful when debugging
		// join behaviour.
		time.Sleep(time.Duration(period) * time.Second)
		log.Printf("Join window has closed after %d seconds.", period)
	}()
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

	waitUntilZStackReady(config.StableFlagFile)

	d.devices = make(map[uint64]*Device)

	conn, err := ninja.Connect("com.ninjablocks.zigbee")
	if err != nil {
		return fmt.Errorf("Could not connect to MQTT: %s", err)
	}

	d.conn = conn

	err = conn.ExportDriver(d)
	if err != nil {
		return fmt.Errorf("Could not export driver: %s", err)
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
		return fmt.Errorf("Error connecting to ota server %s", err)
	}

	d.gatewayConn, err = zigbee.ConnectToGatewayServer(cfg.Hostname, cfg.GatewayPort)
	if err != nil {
		return fmt.Errorf("Error connecting to gateway %s", err)
	}

	/*keyResponse := &nwkmgr.NwkZigbeeGenericCnf{}

	keyRequest := &nwkmgr.NwkChangeNwkKeyReq{
		NewKey: []byte{0x5a, 0x69, 0x67, 0x42, 0x65, 0x65, 0x41, 0x6c, 0x6c, 0x69, 0x61, 0x6e, 0x63, 0x65, 0x30, 0x39},
	}

	err = d.nwkmgrConn.SendCommand(keyRequest, keyResponse)
	if err != nil {
		return fmt.Errorf("Failed setting network key: %s", err)
	}

	spew.Dump(keyResponse)
	log.Println("Sleeping for 10 seconds after setting network key")

	time.Sleep(10 * time.Second)*/

	networkInfo := &nwkmgr.NwkZigbeeNwkInfoCnf{}

	err = d.nwkmgrConn.SendCommand(&nwkmgr.NwkZigbeeNwkInfoReq{}, networkInfo)
	if err != nil {
		return fmt.Errorf("Failed getting network info: %s", err)
	}

	if *networkInfo.Status == nwkmgr.NwkNetworkStatusT_NWK_DOWN {
		return fmt.Errorf("The ZigBee network is down")
	}

	localDevice := &nwkmgr.NwkGetLocalDeviceInfoCnf{}

	err = d.nwkmgrConn.SendCommand(&nwkmgr.NwkGetLocalDeviceInfoReq{}, localDevice)
	if err != nil {
		return fmt.Errorf("Failed getting local device info: %s", err)
	}

	spew.Dump("device info", localDevice.String())

	networkKey := &nwkmgr.NwkGetNwkKeyCnf{}

	err = d.nwkmgrConn.SendCommand(&nwkmgr.NwkGetNwkKeyReq{}, networkKey)
	if err != nil {
		return fmt.Errorf("Failed getting network key: %s", err)
	}

	spew.Dump(networkKey)

	log.Printf("Started coordinator. Channel:%d Pan ID:0x%X Key:% X", *networkInfo.NwkChannel, *networkInfo.PanId, networkKey.NewKey)

	return nil
}

func (d *Driver) FetchDevices() error {

	return d.nwkmgrConn.FetchDeviceList()

}

func (d *Device) GetDriver() ninja.Driver {
	return d.driver
}

func (d *Device) GetDeviceInfo() *model.Device {
	return d.info
}

func (d *Device) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
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

	sigs := make(map[string]string)

	manufacturerName := ""
	modelIdentifier := ""

	for _, attribute := range response.AttributeRecordList {

		switch *attribute.AttributeId {
		case ManufacturerNameAttribute:
			manufacturerName = string(attribute.AttributeValue)
		case ModelIdentifierAttribute:
			modelIdentifier = string(attribute.AttributeValue)
		default:
			log.Printf("Unknown attribute returned when finding basic info %s", *attribute.AttributeId)
		}
	}

	sigs["zigbee:ManufacturerName"] = manufacturerName
	sigs["zigbee:ModelIdentifier"] = modelIdentifier

	name := fmt.Sprintf("%s by %s", modelIdentifier, manufacturerName)
	id := fmt.Sprintf("%X", *d.deviceInfo.IeeeAddress)

	d.info.Name = &name

	d.info.Signatures = &sigs
	d.info.NaturalIDType = "zigbee"
	d.info.NaturalID = id
	d.info.Name = &name

	return nil
}

func (d *Driver) onDeviceFound(deviceInfo *nwkmgr.NwkDeviceInfoT) {

	fmt.Printf("\n\n")
	fmt.Printf("---- Found Device IEEE:%X ----\f", *deviceInfo.IeeeAddress)

	if d.devices[*deviceInfo.IeeeAddress] != nil {
		fmt.Printf("We've already seen this device")
	}

	/*sigs, _ := simplejson.NewJson([]byte(`{
	    "ninja:manufacturer": "Phillips",
	    "ninja:productName": "Hue",
	    "manufacturer:productModelId": "",
	    "ninja:productType": "Light",
	    "ninja:thingType": "light"
	}`))*/

	device := &Device{
		info:       &model.Device{},
		driver:     d,
		deviceInfo: deviceInfo,
	}

	err := device.getBasicInfo()
	if err != nil {
		log.Fatalf("Failed to get basic info: %s", err)
		return
	}

	spew.Dump(deviceInfo)

	err = d.conn.ExportDevice(device)

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
				nil,
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
				nil,
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
				nil,
			}

			err := temp.init()
			if err != nil {
				log.Printf("Failed initialising temp channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDHumidity) {
			log.Printf("This endpoint has humidity cluster")

			humidity := &HumidityChannel{
				Channel{
					device:   device,
					endpoint: endpoint,
				},
				nil,
			}

			err := humidity.init()
			if err != nil {
				log.Printf("Failed initialising humidity channel: %s", err)
			}

		}

	}

	d.devices[*deviceInfo.IeeeAddress] = device

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

func waitUntilZStackReady(checkFile string) {
	if checkFile == "" {
		return
	}

	// cooperate with zigbeeHAgw so that we don't start the zigbee driver
	// until we look somewhat stable.
	log.Printf("waiting until zigbeeHAgw writes %s", checkFile)
	for {
		if _, err := os.Stat(checkFile); err == nil {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
	log.Printf("%s detected. start up continues...", checkFile)
}
