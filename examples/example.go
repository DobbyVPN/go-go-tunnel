// Example usage of the TrustTunnel Go wrapper
package main

import (
	"log"
	"os"
	"time"

	tt "trusttunnel-go/manager"
)

func main() {
	// Create a new TrustTunnelManager instance
	manager := tt.NewTrustTunnelManager()

	// Set up callbacks
	manager.SetStateChangedCallback(func(state tt.VpnState) {
		log.Printf("VPN State changed: %v", state)
	})

	manager.SetLogCallback(func(level tt.LogLevel, message string) {
		log.Printf("[%v] %s", level, message)
	})

	// Optional: Set custom socket protection callback
	// This allows you to implement your own socket protection logic
	// For example, on Android you might call VpnService.protect(fd)
	// On iOS you might bind the socket to a specific interface
	// On Linux you might use SO_BINDTODEVICE
	// On Windows you might use Wintun
	manager.SetProtectSocketCallback(func(fd int) int {
		// Your custom socket protection logic here
		log.Printf("Protecting socket fd: %d", fd)
		return 0 // Return 0 on success, non-zero on failure
	})

	// TrustTunnel TOML configuration
	// This is a minimal example - adjust according to your needs
	// First, try to load config from environment variable
	configStr := os.Getenv("TT_CONFIG")

	// If not in env, try to load from a local file that won't be committed
	if configStr == "" {
		if bytes, err := os.ReadFile("config.toml"); err == nil {
			configStr = string(bytes)
		}
	}

	// If no config provided, use a placeholder
	if configStr == "" {
		log.Println("No TT_CONFIG env var or config.toml found. Using dummy configuration.")
		configStr = `
	loglevel = "info"
	vpn_mode = "general"
	killswitch_enabled = false
	post_quantum_group_enabled = true
	exclusions = []

	[endpoint]
	hostname = "dummy.server.com"
	addresses = ["127.0.0.1:443"]
	custom_sni = "dummy.server.com"
	has_ipv6 = true
	username = "dummy_user"
	password = "dummy_password"
	client_random = ""
	skip_verification = true
	upstream_protocol = "http3"
	anti_dpi = true
	dns_upstreams = []

	[listener]
	[listener.socks]
	address = "127.0.0.1:10808"
	`
	}

	// Start the VPN
	log.Println("Starting VPN...")
	if err := manager.Start(configStr); err != nil {
		log.Fatalf("Failed to start VPN: %v", err)
	}

	// Wait for a while
	log.Println("VPN is running. Press Ctrl+C to stop.")
	time.Sleep(30 * time.Second)

	// Stop the VPN
	log.Println("Stopping VPN...")
	manager.Stop()

	log.Println("VPN stopped.")
}

/*
To create config for your server use trusttunnel's setup_wizard
https://github.com/TrustTunnel/TrustTunnelClient/tree/master/trusttunnel/setup_wizard

./setup_wizard --mode non-interactive `
    --address address:port `
    --hostname server.domain.com `
    --creds user:password `
    --cert cert.pem `
    --settings trusttunnel_client.toml
*/
