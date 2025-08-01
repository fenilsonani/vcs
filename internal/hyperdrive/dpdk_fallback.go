// +build !linux

package hyperdrive

import "errors"

// DPDK fallback for non-Linux systems

type DPDKPort struct{}
type DPDKPacket struct{}
type DPDKStats struct{}

func InitDPDK(args []string) error {
	return errors.New("DPDK not supported on this platform")
}

func NewDPDKPort(portID uint16, numQueues uint16) (*DPDKPort, error) {
	return nil, errors.New("DPDK not supported on this platform")
}

func (p *DPDKPort) Start() error {
	return errors.New("DPDK not supported")
}

func (p *DPDKPort) RecvBurst(queueID uint16, packets []*DPDKPacket) uint16 {
	return 0
}

func (p *DPDKPort) SendBurst(queueID uint16, packets []*DPDKPacket, count uint16) uint16 {
	return 0
}

func (p *DPDKPort) AllocPackets(packets []*DPDKPacket) uint16 {
	return 0
}

func (p *DPDKPort) FreePackets(packets []*DPDKPacket) {
}

func (p *DPDKPort) GetStats() DPDKStats {
	return DPDKStats{}
}