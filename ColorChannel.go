package main

import (
	"fmt"
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type ColorChannel struct {
	Channel
	channel *channels.ColorChannel
}

// -------- Color Protocol --------

func (c *ColorChannel) init() error {
	log.Debugf("Initialising color channel of device %d", *c.device.deviceInfo.IeeeAddress)

	/*clusterID := uint32(0x06)
	  attributeID := uint32(0)
	  minReportInterval := uint32(1)
	  maxReportInterval := uint32(120)

	  request := &gateway.GwSetAttributeReportingReq{
	    DstAddress: &gateway.GwAddressStructT{
	      AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
	      IeeeAddr:    c.device.deviceInfo.IeeeAddress,
	    },
	    ClusterId: &clusterID,
	    AttributeReportList: []*gateway.GwAttributeReportT{{
	      AttributeId:       &attributeID,
	      AttributeType:     gateway.GwZclAttributeDataTypesT_ZCL_DATATYPE_BOOLEAN.Enum(),
	      MinReportInterval: &minReportInterval,
	      MaxReportInterval: &maxReportInterval,
	    }},
	  }

	  response := &gateway.GwSetAttributeReportingRspInd{}

	  err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 20*time.Second)
	  if err != nil {
	    log.Errorf("Error enabling on/off reporting: %s", err)
	  } else if response.Status.String() != "STATUS_SUCCESS" {
	    log.Errorf("Failed to enable on/off reporting. status: %s", response.Status.String())
	  }*/

	//mosquitto_pub -m '{"id":123, "params": [0.1],"jsonrpc": "2.0","method":"set","time":132123123}' -t '$device/26b4f71484/channel/11-8'

	c.channel = channels.NewColorChannel(c)
	err := c.device.driver.Conn.ExportChannel(c.device, c.channel, c.ID)
	if err != nil {
		log.Fatalf("Failed to announce color channel: %s", err)
	}

	go func() {
		for {
			log.Debugf("Polling for color")
			err := c.fetchState()
			if err != nil {
				log.Errorf("Failed to poll for color state %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil

}

func (c *ColorChannel) SetColor(state *channels.ColorState) error {

	if state.Mode != "hue" {
		return fmt.Errorf("TODO: Only color mode 'hue' is supported atm.")
	}

	spew.Dump("setting color", state)

	hue := uint32(*state.Hue * float64(math.MaxUint8-1))
	saturation := uint32(*state.Saturation * float64(math.MaxUint8-1))

	request := &gateway.DevSetColorReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
		HueValue:        &hue,
		SaturationValue: &saturation,
	}

	spew.Dump(request)

	response := &gateway.GwZigbeeGenericRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Error setting color state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to set color state. status: %s", response.Status.String())
	}

	return c.fetchState()
}

func (c *ColorChannel) fetchState() error {
	request := &gateway.DevGetColorReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
	}

	response := &gateway.DevGetColorRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting color state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get color state. status: %s", response.Status.String())
	}

	saturation := float64(float64(*response.SatValue) / float64(math.MaxUint8-1))
	hue := float64(float64(*response.HueValue) / float64(math.MaxUint8-1))

	state := &channels.ColorState{
		Mode:       "Hue",
		Saturation: &saturation,
		Hue:        &hue,
	}

	c.channel.SendState(state)

	return nil
}
