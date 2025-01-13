package network

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"

	"github.com/safepointcloud/safepanel/pkg/models"
)

type IPAnalyzer interface {
	Start(ctx context.Context) error
	Stop() error
	SetNewConnectionCallback(callback func(*models.NewConnectionStats))
	SetDNSQueryCallback(callback func(*models.DNSQueryStats))
	SetDNSResponseCallback(callback func(*models.DNSResponse))
}

type Config struct {
	Interface   string
	BufferSize  int32
	Promiscuous bool
}

type ipAnalyzer struct {
	config   *Config
	handle   *pcapgo.EthernetHandle
	stopChan chan struct{}
	localIPs []net.IP

	// callback
	onNewConnection func(*models.NewConnectionStats)
	onDNSQuery      func(*models.DNSQueryStats)
	onDNSResponse   func(*models.DNSResponse)
}

func NewIPAnalyzer(config *Config) (IPAnalyzer, error) {
	localIPs, err := getLocalIPs()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IPs: %v", err)
	}

	return &ipAnalyzer{
		config:   config,
		localIPs: localIPs,
		stopChan: make(chan struct{}),
	}, nil
}

func (a *ipAnalyzer) Start(ctx context.Context) error {
	handle, err := pcapgo.NewEthernetHandle(a.config.Interface)
	if err != nil {
		return fmt.Errorf("failed to open interface: %v", err)
	}
	a.handle = handle

	// start packet capture
	go a.capture(ctx)

	return nil
}

func (a *ipAnalyzer) capture(ctx context.Context) {
	packetSource := gopacket.NewPacketSource(a.handle, layers.LayerTypeEthernet)
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case packet := <-packetSource.Packets():
			a.processPacket(packet)
		}
	}
}

func (a *ipAnalyzer) processPacket(packet gopacket.Packet) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return
	}

	// handle different protocols
	switch {
	case packet.Layer(layers.LayerTypeTCP) != nil:
		tcp := packet.Layer(layers.LayerTypeTCP).(*layers.TCP)
		a.handleTCPPacket(ip, tcp)
	case packet.Layer(layers.LayerTypeUDP) != nil:
		udp := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)
		a.handleUDPPacket(ip, udp, packet)
	}
}

func (a *ipAnalyzer) handleTCPPacket(ip *layers.IPv4, tcp *layers.TCP) {
	if tcp.SYN && !tcp.ACK {
		direction := models.DirectionOutbound
		for _, localIP := range a.localIPs {
			if ip.DstIP.Equal(localIP) {
				direction = models.DirectionInbound
				break
			}
		}

		newConn := &models.NewConnectionStats{
			SrcIP:     ip.SrcIP.String(),
			SrcPort:   uint16(tcp.SrcPort),
			DstIP:     ip.DstIP.String(),
			DstPort:   uint16(tcp.DstPort),
			Protocol:  models.ProtocolTCP,
			Direction: direction,
			Timestamp: time.Now(),
		}

		if a.onNewConnection != nil {
			a.onNewConnection(newConn)
		}
	}
}

func (a *ipAnalyzer) handleUDPPacket(ip *layers.IPv4, udp *layers.UDP, packet gopacket.Packet) {
	if udp.DstPort == 53 || udp.SrcPort == 53 {
		dnsLayer := packet.Layer(layers.LayerTypeDNS)
		if dnsLayer != nil {
			dns, _ := dnsLayer.(*layers.DNS)

			if !dns.QR { // DNS query
				if a.onDNSQuery != nil {
					for _, question := range dns.Questions {
						query := &models.DNSQueryStats{
							ID:        dns.ID,
							Domain:    string(question.Name),
							SrcIP:     ip.SrcIP.String(),
							DNSServer: ip.DstIP.String(),
							QueryType: question.Type.String(),
							Timestamp: time.Now(),
						}
						a.onDNSQuery(query)
					}
				}
			} else { // DNS response
				if a.onDNSResponse != nil {
					var IPs []string
					for _, answer := range dns.Answers {
						IPs = append(IPs, answer.String())
					}
					if len(IPs) > 0 {
						response := &models.DNSResponse{
							QueryID:   dns.ID,
							Response:  IPs,
							Timestamp: time.Now(),
						}
						a.onDNSResponse(response)
					}
				}
			}
		}
	}
}

func (a *ipAnalyzer) Stop() error {
	if a.handle != nil {
		a.handle.Close()
	}
	close(a.stopChan)
	return nil
}

func (a *ipAnalyzer) SetNewConnectionCallback(callback func(*models.NewConnectionStats)) {
	a.onNewConnection = callback
}

func (a *ipAnalyzer) SetDNSQueryCallback(callback func(*models.DNSQueryStats)) {
	a.onDNSQuery = callback
}

func (a *ipAnalyzer) SetDNSResponseCallback(callback func(*models.DNSResponse)) {
	a.onDNSResponse = callback
}

func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		// skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if ip4 := v.IP.To4(); ip4 != nil {
					ips = append(ips, ip4)
				}
			}
		}
	}
	return ips, nil
}
