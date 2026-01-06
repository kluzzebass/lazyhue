// Package hue provides bridge communication and management.
package hue

import (
	"strings"
	"time"

	"github.com/openhue/openhue-go"
)

// BridgeInfo contains discovered bridge details.
type BridgeInfo struct {
	ID           string // Unique bridge identifier (from mDNS hostname or URL discovery)
	Name         string // Display name (user-assigned after connection, or mDNS instance before)
	IPAddress    string // IP address
	Host         string // mDNS hostname (e.g., "ecb5fa401886.local") - empty for URL discovery
	InstanceName string // mDNS instance name (e.g., "Hue Bridge - 401886")
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

	// Determine the bridge ID from discovery data
	// For mDNS: extract from hostname (e.g., "ecb5fa401886.local" -> "ecb5fa401886")
	// For URL discovery: openhue puts the bridge ID in HostName field
	var bridgeID string
	var hostname string
	var displayName string
	var instanceName string

	if bridge.Instance == "N/A" {
		// URL discovery fallback - HostName contains the bridge ID
		bridgeID = bridge.HostName
		hostname = ""                            // No mDNS hostname available
		displayName = "Bridge " + bridge.HostName // Use bridge ID as display name
		instanceName = ""                        // No mDNS instance
	} else {
		// mDNS discovery - HostName is the actual .local hostname
		hostname = bridge.HostName
		// Extract ID from hostname (strip .local suffix)
		bridgeID = strings.TrimSuffix(bridge.HostName, ".local")
		bridgeID = strings.TrimSuffix(bridgeID, ".") // Some systems include trailing dot
		displayName = bridge.Instance
		instanceName = bridge.Instance
	}

	info := BridgeInfo{
		ID:           bridgeID,
		Name:         displayName, // Will be replaced with user name after connection
		IPAddress:    bridge.IpAddress,
		Host:         hostname,
		InstanceName: instanceName,
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

