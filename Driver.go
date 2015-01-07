package main

import (
	"fmt"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/events"
	"github.com/ninjasphere/go-ninja/logger"
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
	ClusterIDIASZone  uint32 = 0x500
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

	localDevice *nwkmgr.NwkDeviceInfoT
	devices     map[uint64]*Device

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
	log = driver.Log
	zigbee.SetLogger(logger.GetLogger(info.ID + ".backend"))

	err = driver.Export(driver)
	if err != nil {
		log.Fatalf("Failed to export fake driver: %s", err)
	}

	userAgent := driver.Conn.GetServiceClient("$device/:deviceId/channel/user-agent")
	userAgent.OnEvent("pairing-requested", func(pairingRequest *events.PairingRequest, values map[string]string) bool {

		duration := uint32(pairingRequest.Duration)

		if duration > 254 {
			duration = 254
		}

		log.Infof("Pairing request received from %s for %d seconds", values["deviceId"], duration)

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
	go func() {
		// startup can take a while, so always succeed but then die if it fails.

		// required because inbound RPC start calls have arbitrary timeouts which a) we are not aware of
		// and b) we can't guarantee to satisfy here.

		err := d.startup()
		if err != nil {
			d.Log.Fatalf("startup failed : %v", err)
			os.Exit(1)
		}
	}()
	return nil
}

func (d *Driver) startup() error {

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

	d.localDevice = localDevice.DeviceInfoList

	if log.IsDebugEnabled() {
		spew.Dump("device info", localDevice.String())
	}

	/*networkKey := &nwkmgr.NwkGetNwkKeyCnf{}

	err = d.nwkmgrConn.SendCommand(&nwkmgr.NwkGetNwkKeyReq{}, networkKey)
	if err != nil {
		log.Fatalf("Failed getting network key: %s", err)
	}

	if log.IsDebugEnabled() {
		spew.Dump(networkKey)
	}*/

	log.Debugf("Started coordinator. Channel:%d Pan ID:0x%X", *networkInfo.NwkChannel, *networkInfo.PanId)

	d.StartFetchingDevices()

	return nil
}

func (d *Driver) StartPairing(period uint32) (*uint32, error) {
	if period > 254 {
		period = 254
	}
	err := d.EnableJoin(period)
	return &period, err
}

func (d *Driver) EndPairing() error {
	err := d.EnableJoin(0)
	return err
}

func (d *Driver) EnableJoin(duration uint32) error {

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

	go func() {
		save := d.devicesFound
		time.Sleep(time.Second * time.Duration(duration))
		d.Log.Infof("Join window closes after %d seconds.", duration)
		d.SendEvent("pairing-ended", &events.PairingEnded{
			DevicesFound: int(d.devicesFound - save),
		})
	}()

	d.Log.Infof("Join window opens for %d seconds", duration)
	return nil
}

func (d *Driver) StartFetchingDevices() {
	go func() {
		for {
			d.nwkmgrConn.FetchDeviceList()
			time.Sleep(time.Second * 30)
		}
	}()
}

func (d *Driver) onDeviceFound(deviceInfo *nwkmgr.NwkDeviceInfoT) {

	// XXX: This device status is often wrong. Useless.
	/*if *deviceInfo.DeviceStatus != nwkmgr.NwkDeviceStatusT_DEVICE_ON_LINE {
		log.Debugf("---- Found Offline Device IEEE:%X ----\f", *deviceInfo.IeeeAddress)

		// TODO: Inform the device (if we've seen it online) to stop polling

		return
	}*/

	if d.devices[*deviceInfo.IeeeAddress] != nil {
		// We've seen this already, but it may have been re-paired. We *should* just be able to replace
		// the deviceInfo object, which is used for all communication.
		// TODO: Actually verify this. May need to re-run channel init.
		d.devices[*deviceInfo.IeeeAddress].deviceInfo = deviceInfo
		return
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

		if *endpoint.ProfileId == 0x104 /* HA */ {
			switch *endpoint.DeviceId {
			case 0x009: // Mains Power Outlet
				(*device.info.Signatures)["ninja:thingType"] = "socket"
			}
		}
	}

	err := device.getBasicInfo()
	if err != nil {
		log.Debugf("Failed to get basic info for: %X : %s", *deviceInfo.IeeeAddress, err)
		return
	}

	name := ""

	if device.ModelIdentifier != "" {
		(*device.info.Signatures)["zigbee:ModelIdentifier"] = device.ModelIdentifier
		name = device.ModelIdentifier
	}

	if device.ManufacturerName != "" {
		(*device.info.Signatures)["zigbee:ManufacturerName"] = device.ManufacturerName
		if device.ModelIdentifier != "" {
			name += " by "
		}
		name += device.ManufacturerName
	}

	log.Debugf("\n\n")
	log.Infof("---- Found Device IEEE:%X Name:%s ----\f", *deviceInfo.IeeeAddress, name)
	log.Debugf("Device Info: %v", *deviceInfo)

	if name != "" {
		device.info.Name = &name
	}

	if log.IsDebugEnabled() {
		spew.Dump(deviceInfo)
	}

	err = d.Conn.ExportDevice(device)
	if err != nil {
		log.Fatalf("Failed to export zigbee device %s: %s", name, err)
	}
	d.devicesFound++

	d.devices[*deviceInfo.IeeeAddress] = device

	batchChannel := &BatchChannel{
		Channel: Channel{
			ID:     "batch",
			device: device,
		},
	}

	log.Debugf("Got device : %d", *deviceInfo.IeeeAddress)

	for _, endpoint := range deviceInfo.SimpleDescList {
		log.Debugf("Got endpoint : %d", *endpoint.EndpointId)

		if containsUInt32(endpoint.InputClusters, ClusterIDOnOff) {
			log.Debugf("This endpoint has an input on/off cluster")

			onOff := &OnOffChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d-in", *endpoint.EndpointId, ClusterIDOnOff),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := onOff.init()
			if err != nil {
				log.Debugf("Failed initialising input on/off channel: %s", err)
			}

			batchChannel.onOff = onOff
		}

		if containsUInt32(endpoint.OutputClusters, ClusterIDOnOff) {
			log.Debugf("This endpoint has an output on/off cluster")

			onOff := &OnOffSwitchCluster{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d-out", *endpoint.EndpointId, ClusterIDOnOff),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := onOff.init()
			if err != nil {
				log.Debugf("Failed initialising output on/off channel: %s", err)
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

			batchChannel.brightness = brightness
		}

		if containsUInt32(endpoint.InputClusters, ClusterIDColor) {
			log.Debugf("This endpoint has color cluster.")

			if log.IsDebugEnabled() {
				spew.Dump("color cluster", endpoint, ClusterIDColor)
			}

			color := &ColorChannel{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDColor),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := color.init()
			if err != nil {
				log.Debugf("Failed initialising color channel: %s", err)
			}

			batchChannel.color = color
		}

		if containsUInt32(endpoint.InputClusters, ClusterIDIASZone) {
			log.Debugf("This endpoint has IAS Zone cluster.")

			if log.IsDebugEnabled() {
				spew.Dump("ias zone cluster", endpoint, ClusterIDIASZone)
			}

			color := &IASZoneCluster{
				Channel: Channel{
					ID:       fmt.Sprintf("%d-%d", *endpoint.EndpointId, ClusterIDIASZone),
					device:   device,
					endpoint: endpoint,
				},
			}

			err := color.init()
			if err != nil {
				log.Debugf("Failed initialising IAS Zone channel: %s", err)
			}

		}

	}

	if batchChannel.brightness != nil || batchChannel.color != nil {
		if err := batchChannel.init(); err != nil {
			log.Warningf("Failed to export batch channel: %s", err)
		}
	}

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
