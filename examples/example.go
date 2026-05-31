// Example usage of the TrustTunnel Go wrapper
package main

import (
	"log"
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
	config := `
[server]
address = "your_server_address"
port = 443

[credentials]
username = "dobby_user"
password = "dobby_password"

[listener]
type = "socks"
address = "127.0.0.1:1080"

[upstream]
protocol = "http2"
location = "us-east"
`

	// Start the VPN
	log.Println("Starting VPN...")
	if err := manager.Start(config); err != nil {
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
