package main

import (
	"fmt"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/events"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-ninja/support"
	"github.com/ninjasphere/go-zigbee"
	"github.com/ninjasphere/go-zigbee/nwkmgr"
)

const (
	ClusterIDBasic    uint32 = 0x00
	ClusterIDOnOff    uint32 = 0x06
	ClusterIDLevel    uint32 = 0x08 // We're always exporting as brightness for now
	ClusterIDColor    uint32 = 0x300
	ClusterIDTemp     uint32 = 0x402
	ClusterIDHumidity uint32 = 0x405
	ClusterIDPower    uint32 = 0x702
)

type ZStackConfig struct {
	Hostname       string
	OtasrvrPort    int
	GatewayPort    int
	NwkmgrPort     int
	StableFlagFile string
}

type Driver struct {
	support.DriverSupport

	devices map[uint64]*Device

	config *ZStackConfig

	nwkmgrConn  *zigbee.ZStackNwkMgr
	gatewayConn *zigbee.ZStackGateway
	otaConn     *zigbee.ZStackOta

	devicesFound int
}

func NewDriver(config *ZStackConfig) (*Driver, error) {
	driver := &Driver{
		config:  config,
		devices: make(map[uint64]*Device),
	}

	err := driver.Init(info)
	if err != nil {
		log.Fatalf("Failed to initialize fake driver: %s", err)
	}

	err = driver.Export(driver)
	if err != nil {
		log.Fatalf("Failed to export fake driver: %s", err)
	}

	userAgent := driver.Conn.GetServiceClient("$device/:deviceId/channel/user-agent")
	userAgent.OnEvent("pairing-requested", func(pairingRequest *events.PairingRequest, values map[string]string) bool {
		log.Infof("Pairing request received from %s for %d seconds", values["deviceId"], pairingRequest.Duration)

		duration := uint32(pairingRequest.Duration)

		if err := driver.EnableJoin(duration); err != nil {
			log.Warningf("Failed to enable joining: %s", err)
		}

		return true
	})

	return driver, nil

}

func (d *Driver) Reset(hard bool) error {
	return d.nwkmgrConn.Reset(hard)
}

func (d *Driver) Start() error {

	waitUntilZStackReady(d.config.StableFlagFile)

	var err error

	d.nwkmgrConn, err = zigbee.ConnectToNwkMgrServer(d.config.Hostname, d.config.NwkmgrPort)
	if err != nil {
		return fmt.Errorf("Error connecting to nwkmgr %s", err)
	}
	d.nwkmgrConn.OnDeviceFound = func(deviceInfo *nwkmgr.NwkDeviceInfoT) {
		d.onDeviceFound(deviceInfo)
	}

	/*done := false
	d.nwkmgrConn.OnNetworkReady = func() {
		if !done {
			done = true
		}
	}*/

	d.otaConn, err = zigbee.ConnectToOtaServer(d.config.Hostname, d.config.OtasrvrPort)
	if err != nil {
		return fmt.Errorf("Error connecting to ota server %s", err)
	}

	d.gatewayConn, err = zigbee.ConnectToGatewayServer(d.config.Hostname, d.config.GatewayPort)
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
		if log.IsDebugEnabled() {
			spew.Dump(networkInfo)
		}
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

	if log.IsDebugEnabled() {
		spew.Dump("device info", localDevice.String())
	}

	networkKey := &nwkmgr.NwkGetNwkKeyCnf{}

	err = d.nwkmgrConn.SendCommand(&nwkmgr.NwkGetNwkKeyReq{}, networkKey)
	if err != nil {
		log.Fatalf("Failed getting network key: %s", err)
	}

	if log.IsDebugEnabled() {
		spew.Dump(networkKey)
	}

	log.Debugf("Started coordinator. Channel:%d Pan ID:0x%X Key:% X", *networkInfo.NwkChannel, *networkInfo.PanId, networkKey.NewKey)

	return d.FetchDevices()
}

func (d *Driver) EnableJoin(duration uint32) error {

	go func() {
	   	save := d.devicesFound;
		time.Sleep(time.Second * time.Duration(duration))
		d.Log.Infof("Join window closes after %d seconds.", duration);
		d.SendEvent("pairing-ended", &events.PairingEnded{
			DevicesFound: int(d.devicesFound - save),
		})
	}()

	permitJoinRequest := &nwkmgr.NwkSetPermitJoinReq{
		PermitJoinTime: &duration,
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

	d.SendEvent("pairing-started", &events.PairingStarted{
		Duration: int(duration),
	})

	d.Log.Infof("Join window opens for %d seconds", duration);

	return nil
}

func (d *Driver) FetchDevices() error {
	return d.nwkmgrConn.FetchDeviceList()
}

func (d *Driver) onDeviceFound(deviceInfo *nwkmgr.NwkDeviceInfoT) {

	log.Debugf("\n\n")
	log.Debugf("---- Found Device IEEE:%X ----\f", *deviceInfo.IeeeAddress)
	log.Debugf("Device Info: %v", *deviceInfo)

	if d.devices[*deviceInfo.IeeeAddress] != nil {
		// We've seen this already, but it may have been repaired. We *should* just be able to replace
		// the deviceInfo object, which is used for all communication.
		// TODO: Actually verify this. May need to re-run channel init.
		d.devices[*deviceInfo.IeeeAddress].deviceInfo = deviceInfo
	}

	device := &Device{
		driver:     d,
		deviceInfo: deviceInfo,
		info: &model.Device{
			NaturalID:     fmt.Sprintf("%X", *deviceInfo.IeeeAddress),
			NaturalIDType: "zigbee",
			Signatures:    &map[string]string{},
		},
	}

	for _, endpoint := range deviceInfo.SimpleDescList {
		if *endpoint.ProfileId == 0xC05E /*ZLL*/ {

			switch *endpoint.DeviceId {

			case 0x0000: // On/Off Light
				fallthrough
			case 0x0100: // Dimmable Light
				fallthrough
			case 0x0200: // Color Light
				fallthrough
			case 0x210: // Ext Color Light
				(*device.info.Signatures)["ninja:thingType"] = "light"
			}
		}
	}

	err := device.getBasicInfo()
	if err != nil {
		log.Debugf("Failed to get basic info: %s", err)
	}

	if device.ManufacturerName != "" {
		(*device.info.Signatures)["zigbee:ManufacturerName"] = device.ManufacturerName
	} else {
		device.ManufacturerName = "Unknown"
	}
	if device.ModelIdentifier != "" {
		(*device.info.Signatures)["zigbee:ModelIdentifier"] = device.ModelIdentifier
	} else {
		device.ModelIdentifier = fmt.Sprintf("MAC:%X", *deviceInfo.IeeeAddress)
	}

	name := fmt.Sprintf("%s by %s", device.ModelIdentifier, device.ManufacturerName)

	device.info.Name = &name

	if log.IsDebugEnabled() {
		spew.Dump(deviceInfo)
	}

	err = d.Conn.ExportDevice(device)
	if err != nil {
		log.Fatalf("Failed to export zigbee device %s: %s", name, err)
	}
	d.devicesFound++;

	log.Debugf("Got device : %d", *deviceInfo.IeeeAddress)

	for _, endpoint := range deviceInfo.SimpleDescList {
		log.Debugf("Got endpoint : %d", *endpoint.EndpointId)

		if containsUInt32(endpoint.InputClusters, ClusterIDOnOff) {
			log.Debugf("This endpoint has on/off cluster")

			onOff := &OnOffChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDOnOff),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := onOff.init()
			if err != nil {
				log.Debugf("Failed initialising on/off channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDPower) {
			log.Debugf("This endpoint has power cluster")

			power := &PowerChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDPower),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := power.init()
			if err != nil {
				log.Debugf("Failed initialising power channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDTemp) {
			log.Debugf("This endpoint has temperature cluster")

			temp := &TempChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDTemp),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := temp.init()
			if err != nil {
				log.Debugf("Failed initialising temp channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDHumidity) {
			log.Debugf("This endpoint has humidity cluster")

			humidity := &HumidityChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDHumidity),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := humidity.init()
			if err != nil {
				log.Debugf("Failed initialising humidity channel: %s", err)
			}

		}

		if containsUInt32(endpoint.InputClusters, ClusterIDLevel) {
			log.Debugf("This endpoint has level cluster. Exporting as brightness channel")

			if log.IsDebugEnabled() {
				spew.Dump("brightness cluster", endpoint, ClusterIDLevel)
			}

			brightness := &BrightnessChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDLevel),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := brightness.init()
			if err != nil {
				log.Debugf("Failed initialising brightness channel: %s", err)
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
	log.Debugf("waiting until zigbeeHAgw writes %s", checkFile)
	for {
		if _, err := os.Stat(checkFile); err == nil {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
	log.Debugf("%s detected. start up continues...", checkFile)
}
