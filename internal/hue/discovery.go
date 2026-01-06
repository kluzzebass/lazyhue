// Package hue provides bridge communication and management.
package hue

import (
	"time"

	"github.com/openhue/openhue-go"
)

// BridgeInfo contains discovered bridge details.
type BridgeInfo struct {
	ID        string
	Name      string
	IPAddress string
	Host      string
}

// DiscoveryService handles finding Hue bridges on the network.
type DiscoveryService struct {
	timeout time.Duration
}

// NewDiscoveryService creates a discovery service with the given timeout.
func NewDiscoveryService(timeout time.Duration) *DiscoveryService {
	return &DiscoveryService{timeout: timeout}
}

// Discover finds Hue bridges on the local network.
// It first tries mDNS, then falls back to discovery.meethue.com.
func (d *DiscoveryService) Discover() ([]BridgeInfo, error) {
	discovery := openhue.NewBridgeDiscovery(openhue.WithTimeout(d.timeout))
	
	bridge, err := discovery.Discover()
	if err != nil {
		return nil, err
	}

	// openhue-go currently returns a single bridge
	// We wrap it to support multiple bridges in the future
	info := BridgeInfo{
		ID:        extractBridgeID(bridge.IpAddress),
		Name:      bridge.Instance,
		IPAddress: bridge.IpAddress,
		Host:      bridge.IpAddress,
	}

	return []BridgeInfo{info}, nil
}

// DiscoverAll attempts to find all bridges, continuing on partial failures.
func (d *DiscoveryService) DiscoverAll() []BridgeInfo {
	bridges, err := d.Discover()
	if err != nil {
		return []BridgeInfo{}
	}
	return bridges
}

// extractBridgeID derives a bridge ID from the hostname.
// Typically the hostname is like "ecb5fa1a3e4f.local."
func extractBridgeID(host string) string {
	if len(host) > 0 {
		// Strip .local. suffix if present
		id := host
		if len(id) > 7 && id[len(id)-7:] == ".local." {
			id = id[:len(id)-7]
		}
		return id
	}
	return "unknown"
}

