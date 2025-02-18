/*
Copyright 2020 The Magma Authors.
This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree.
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package servicers

import (
	"fmt"
	"net"

	oaiprotos "magma/lte/cloud/go/protos/oai"

	"github.com/golang/glog"
	"github.com/wmnsk/go-gtp/gtpv2/ie"
)

func handleTFT(tftIE *ie.IE) (*oaiprotos.TrafficFlowTemplate, error) {
	tftFieldsIE, err := tftIE.BearerTFT()
	if err != nil {
		err = fmt.Errorf("Couldn't get Bearer TFT: %s ", err)
		return nil, err
	}

	// generate list of packet filters
	packetFilters, err := handlePacketFilters(tftFieldsIE)
	if err != nil {
		return nil, err
	}

	// decide kind of packet filter
	tftPacketFilterList := &oaiprotos.PacketFilterList{}
	switch tftFieldsIE.OperationCode {
	case ie.TFTOpCreateNewTFT:
		tftPacketFilterList.CreateNewTft = packetFilters
	case ie.TFTOpAddPacketFiltersToExistingTFT:
		tftPacketFilterList.AddPacketFilter = packetFilters
	case ie.TFTOpReplacePacketFiltersInExistingTFT:
		tftPacketFilterList.ReplacePacketFilter = packetFilters
	case ie.TFTOpDeleteExistingTFT:
		return nil, fmt.Errorf("TFTOpDeleteExistingTFT Unimplemented")
	}

	tft := &oaiprotos.TrafficFlowTemplate{
		PacketFilterList: tftPacketFilterList,
		ParametersList:   nil,
		TftOperationCode: uint32(tftFieldsIE.OperationCode),
	}

	return tft, nil
}

func handlePacketFilters(tftFieldsIE *ie.TrafficFlowTemplate) ([]*oaiprotos.PacketFilter, error) {
	packetFilters := []*oaiprotos.PacketFilter{}
	for _, tftPacketFilterIE := range tftFieldsIE.PacketFilters {
		for _, packetComponentIE := range tftPacketFilterIE.Components {
			component, err := handlePacketFilterComponent(packetComponentIE)
			if err != nil {
				glog.Infof("Couldn't parse Packet Filter Component: %s ", err)
				continue
			}
			packetFilter := &oaiprotos.PacketFilter{
				Spare:                0,
				Direction:            uint32(tftPacketFilterIE.Direction),
				Identifier:           uint32(tftPacketFilterIE.Identifier),
				EvalPrecedence:       uint32(tftPacketFilterIE.EvaluationPrecedence),
				Length:               uint32(tftPacketFilterIE.Length),
				PacketFilterContents: component,
			}

			packetFilters = append(packetFilters, packetFilter)
		}
	}
	return packetFilters, nil
}

func handlePacketFilterComponent(packetComponentIE *ie.TFTPFComponent) (*oaiprotos.PacketFilterContents, error) {
	content := &oaiprotos.PacketFilterContents{
		Ipv4RemoteAddresses: make([]*oaiprotos.IpRemoteAddress, 0),
		Ipv6RemoteAddresses: make([]*oaiprotos.IpRemoteAddress, 0),
	}
	content.Flags = uint32(packetComponentIE.Type)
	switch packetComponentIE.Type {
	case ie.PFCompIPv4RemoteAddress:
		ipv4, err := packetComponentIE.IPv4RemoteAddress()
		if err != nil {
			return nil, fmt.Errorf("PFCompIPv4RemoteAddress %s", err)
		}
		content.Ipv4RemoteAddresses = append(content.Ipv4RemoteAddresses,
			&oaiprotos.IpRemoteAddress{
				Addr: ip2Long(ipv4.IP.String()),
				Mask: ip2Long(net.IP(ipv4.Mask).String()),
			},
		)
	case ie.PFCompIPv6RemoteAddress:
		ipv6, err := packetComponentIE.IPv6RemoteAddress()
		if err != nil {
			return nil, fmt.Errorf("PFCompIPv6RemoteAddress %s", err)
		}
		content.Ipv6RemoteAddresses = append(content.Ipv6RemoteAddresses,
			&oaiprotos.IpRemoteAddress{
				Addr: ip2Long(ipv6.IP.String()),
				Mask: ip2Long(net.IP(ipv6.Mask).String()),
			},
		)
	case ie.PFCompProtocolIdentifierNextHeader:
		protocolId, err := packetComponentIE.ProtocolIdentifierNextHeader()
		if err != nil {
			return nil, fmt.Errorf("PFCompProtocolIdentifierNextHeader %s", err)
		}
		content.ProtocolIdentifierNextheader = uint32(protocolId)
	case ie.PFCompSingleLocalPort:
		port, err := packetComponentIE.SingleLocalPort()
		if err != nil {
			return nil, fmt.Errorf("PFCompSingleLocalPort %s", err)
		}
		content.SingleLocalPort = uint32(port)

	case ie.PFCompSingleRemotePort:
		port, err := packetComponentIE.SingleRemotePort()
		if err != nil {
			return nil, fmt.Errorf("PFCompSingleRemotePort %s", err)
		}
		content.SingleRemotePort = uint32(port)

	case ie.PFCompLocalPortRange:
		start, end, err := packetComponentIE.LocalPortRange()
		if err != nil {
			return nil, fmt.Errorf("PFCompLocalPortRange %s", err)
		}
		content.LocalPortRange = &oaiprotos.PortRange{
			LowLimit:  uint32(start),
			HighLimit: uint32(end),
		}
	case ie.PFCompRemotePortRange:
		start, end, err := packetComponentIE.RemotePortRange()
		if err != nil {
			return nil, fmt.Errorf("PFCompRemotePortRange %s", err)
		}
		content.RemotePortRange = &oaiprotos.PortRange{
			LowLimit:  uint32(start),
			HighLimit: uint32(end),
		}
	case ie.PFCompSecurityParameterIndex:
		idx, err := packetComponentIE.SecurityParameterIndex()
		if err != nil {
			return nil, fmt.Errorf("PFCompSecurityParameterIndex %s", err)
		}
		content.SecurityParameterIndex = idx
	case ie.PFCompTypeOfServiceTrafficClass:
		value, mask, err := packetComponentIE.TypeOfServiceTrafficClass()
		if err != nil {
			return nil, fmt.Errorf("PFCompTypeOfServiceTrafficClass %s", err)
		}
		content.TypeOfServiceTrafficClass = &oaiprotos.TypeOfServiceTrafficClass{
			Value: uint32(value),
			Mask:  uint32(mask),
		}
	}

	return content, nil
}
