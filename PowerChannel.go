package main

import (
	"log"

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

	reportingRequest := &gateway.GwSetAttributeReportingReq{
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

	confirmation := &gateway.GwZigbeeGenericCnf{}

	err := c.device.driver.gatewayConn.SendCommand(reportingRequest, confirmation)
	if err != nil {
		log.Fatalf("Failed to enable reporting on power device: %s", err)
	}
	if confirmation.Status.String() != "STATUS_SUCCESS" {
		log.Fatalf("Failed to enable reporting on power device. Status:%s", confirmation.Status.String())
	}

	reportingResponse := &gateway.GwSetAttributeReportingRspInd{}
	err = c.device.driver.gatewayConn.WaitForSequenceResponse(confirmation.SequenceNumber, reportingResponse)
	if err != nil {
		log.Fatalf("Failed to get power reporting response: %s", err)
	}

	return nil
}
