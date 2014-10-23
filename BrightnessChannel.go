package main

import (
	"fmt"
	"math"
	"time"

	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type BrightnessChannel struct {
	Channel
	channel *channels.BrightnessChannel
}

// -------- Brightness Protocol --------

func (c *BrightnessChannel) init() error {
	log.Debugf("Initialising brightness channel of device %d", *c.device.deviceInfo.IeeeAddress)

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

	c.channel = channels.NewBrightnessChannel(c)
	err := c.device.driver.Conn.ExportChannel(c.device, c.channel, c.ID)
	if err != nil {
		log.Fatalf("Failed to announce brightness channel: %s", err)
	}

	go func() {
		for {
			log.Debugf("Polling for brightness")
			err := c.fetchState()
			if err != nil {
				log.Errorf("Failed to poll for brightness state %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil

}

func (c *BrightnessChannel) SetBrightness(state float64) error {
	level := uint32(state * float64(math.MaxUint8))
	transition := uint32(1) // 1/10th seconds?

	request := &gateway.DevSetLevelReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
		LevelValue:     &level,
		TransitionTime: &transition,
	}

	response := &gateway.GwZigbeeGenericRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Error setting brightness state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to set brightness state. status: %s", response.Status.String())
	}

	return c.fetchState()
}

func (c *BrightnessChannel) fetchState() error {
	request := &gateway.DevGetLevelReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
	}

	response := &gateway.DevGetLevelRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting brightness state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get brightness state. status: %s", response.Status.String())
	}

	c.channel.SendState(float64(*response.LevelValue) / float64(math.MaxUint8))

	return nil
}
