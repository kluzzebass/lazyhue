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
		ID:        extractBridgeID(bridge.Instance),
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

// extractBridgeID derives a stable bridge ID from the instance name.
// Instance is typically "Hue Bridge - AABBCC" where AABBCC is part of the MAC.
func extractBridgeID(instance string) string {
	// Try to extract the hex ID from "Hue Bridge - AABBCC" format
	if len(instance) > 13 {
		// Look for the last space-separated part
		for i := len(instance) - 1; i >= 0; i-- {
			if instance[i] == ' ' {
				candidate := instance[i+1:]
				if len(candidate) >= 6 {
					return candidate
				}
				break
			}
		}
	}
	// Fallback: use the whole instance name as ID
	if len(instance) > 0 {
		return instance
	}
	return "unknown"
}

