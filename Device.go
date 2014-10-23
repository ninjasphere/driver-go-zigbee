package main

import (
	"fmt"
	"time"

	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go-zigbee/gateway"
	"github.com/ninjasphere/go-zigbee/nwkmgr"
)

type Device struct {
	info      *model.Device
	sendEvent func(event string, payload interface{}) error

	ManufacturerName string
	ModelIdentifier  string

	driver     *Driver
	deviceInfo *nwkmgr.NwkDeviceInfoT
	channels   []Channel
}

func (d *Device) getBasicInfo() error {

	log.Debugf("Getting basic information from %X", *d.deviceInfo.IeeeAddress)

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
			log.Debugf("Unknown attribute returned when finding basic info %s", *attribute.AttributeId)
		}
	}

	return nil
}

func (d *Device) GetDeviceInfo() *model.Device {
	return d.info
}

func (d *Device) GetDriver() ninja.Driver {
	return d.driver
}

func (d *Device) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}
