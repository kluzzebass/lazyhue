// Package hue provides bridge communication and management.
package hue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
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

// Discover finds Hue bridges on the local network using mDNS.
func (d *DiscoveryService) Discover() ([]BridgeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	// Hue bridges advertise as _hue._tcp
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry, 10)
	var bridges []BridgeInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range entries {
			if len(entry.AddrIPv4) == 0 {
				continue
			}
			ipAddr := entry.AddrIPv4[0].String()

			// Get the real bridge ID and name from the unauthenticated config endpoint
			config, err := fetchBridgeConfig(ipAddr)
			if err != nil {
				// Fallback if config fetch fails
				mu.Lock()
				bridges = append(bridges, BridgeInfo{
					ID:        ipAddr, // Use IP as fallback ID
					Name:      entry.Instance,
					IPAddress: ipAddr,
				})
				mu.Unlock()
				continue
			}

			mu.Lock()
			bridges = append(bridges, BridgeInfo{
				ID:        config.BridgeID,
				Name:      config.Name,
				IPAddress: ipAddr,
			})
			mu.Unlock()
		}
	}()

	// Browse starts the mDNS lookup. It closes the entries channel when ctx is done.
	err = resolver.Browse(ctx, "_hue._tcp", "local.", entries)
	if err != nil {
		return nil, fmt.Errorf("mDNS browse failed: %w", err)
	}

	// Wait for context timeout - zeroconf will close the entries channel
	<-ctx.Done()
	wg.Wait()

	return bridges, nil
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
