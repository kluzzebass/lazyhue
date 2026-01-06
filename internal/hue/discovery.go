// Package hue provides bridge communication and management.
package hue

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/openhue/openhue-go"
)

// BridgeInfo contains discovered bridge details.
type BridgeInfo struct {
	ID        string // Unique bridge hardware ID (e.g., "001788FFFE401886")
	Name      string // User-assigned bridge name
	IPAddress string // IP address
}

// bridgeConfig is the response from /api/0/config (unauthenticated)
type bridgeConfig struct {
	Name     string `json:"name"`
	BridgeID string `json:"bridgeid"`
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
func (d *DiscoveryService) Discover() ([]BridgeInfo, error) {
	discovery := openhue.NewBridgeDiscovery(openhue.WithTimeout(d.timeout))

	bridge, err := discovery.Discover()
	if err != nil {
		return nil, err
	}

	// Get the real bridge ID and name from the unauthenticated config endpoint
	config, err := fetchBridgeConfig(bridge.IpAddress)
	if err != nil {
		// Fallback if config fetch fails
		return []BridgeInfo{{
			ID:        bridge.IpAddress, // Use IP as fallback ID
			Name:      bridge.IpAddress,
			IPAddress: bridge.IpAddress,
		}}, nil
	}

	return []BridgeInfo{{
		ID:        config.BridgeID,
		Name:      config.Name,
		IPAddress: bridge.IpAddress,
	}}, nil
}

// fetchBridgeConfig gets bridge info from the unauthenticated config endpoint.
func fetchBridgeConfig(ipAddress string) (*bridgeConfig, error) {
	// Hue bridges use self-signed certs
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := fmt.Sprintf("https://%s/api/0/config", ipAddress)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config endpoint returned %d", resp.StatusCode)
	}

	var config bridgeConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// DiscoverAll attempts to find all bridges, continuing on partial failures.
func (d *DiscoveryService) DiscoverAll() []BridgeInfo {
	bridges, err := d.Discover()
	if err != nil {
		return []BridgeInfo{}
	}
	return bridges
}
