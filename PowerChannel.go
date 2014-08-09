package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ninjasphere/go-zigbee/gateway"
)

type PowerChannel struct {
	Channel
}

func (c *PowerChannel) init() error {
	log.Printf("Initialising power channel of device %d", *c.device.deviceInfo.IeeeAddress)

	clusterID := ClusterIDPower
	instantaneousDemandAttributeID := uint32(0x0400)
	minReportInterval := uint32(1)
	maxReportInterval := uint32(120)
	reportableChange := uint32(1)

	request := &gateway.GwSetAttributeReportingReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
			EndpointId:  c.endpoint.EndpointId,
		},
		ClusterId: &clusterID,
		AttributeReportList: []*gateway.GwAttributeReportT{&gateway.GwAttributeReportT{
			AttributeId:       &instantaneousDemandAttributeID,
			AttributeType:     gateway.GwZclAttributeDataTypesT_ZCL_DATATYPE_INT24.Enum(),
			MinReportInterval: &minReportInterval,
			MaxReportInterval: &maxReportInterval,
			ReportableChange:  &reportableChange,
		}},
	}

	response := &gateway.GwSetAttributeReportingRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Error enabling power reporting: %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to enable power reporting. status: %s", response.Status.String())
	}

	return nil
}
