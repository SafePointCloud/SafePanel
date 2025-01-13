package network

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/safepointcloud/safepanel/pkg/models"
)

type DNSAnalyzer struct {
	collector *models.StatsCollector
}

func NewDNSAnalyzer() *DNSAnalyzer {
	return &DNSAnalyzer{
		collector: models.NewStatsCollector(),
	}
}

func (d *DNSAnalyzer) ProcessPacket(packet gopacket.Packet) {
	// 获取IP层
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}
	ip, _ := ipLayer.(*layers.IPv4)

	// 获取UDP层
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return
	}
	udp, _ := udpLayer.(*layers.UDP)

	// 检查是否是DNS查询（默认端口53）
	if udp.DstPort != 53 {
		return
	}

	// 获取DNS层
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		return
	}
	dns, _ := dnsLayer.(*layers.DNS)

	// 只处理DNS查询
	if !dns.QR {
		for _, question := range dns.Questions {
			d.collector.AddDNSQuery(&models.DNSQueryStats{
				Domain:    string(question.Name),
				SrcIP:     ip.SrcIP.String(),
				DNSServer: ip.DstIP.String(),
				QueryType: question.Type.String(),
				Timestamp: time.Now(),
			})
		}
	}
}
