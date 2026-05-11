// Example usage of the DobbyBridge Go wrapper
package main

import (
	"log"
	"time"

	"github.com/yourusername/trusttunnel-go/dobby_bridge"
)

func main() {
	// Create a new DobbyBridge instance
	bridge := dobby_bridge.NewDobbyBridge()

	// Set up callbacks
	bridge.SetStateChangedCallback(func(state dobby_bridge.VpnState) {
		log.Printf("VPN State changed: %v", state)
	})

	bridge.SetLogCallback(func(level dobby_bridge.LogLevel, message string) {
		log.Printf("[%v] %s", level, message)
	})

	// Optional: Set custom socket protection callback
	// This allows you to implement your own socket protection logic
	bridge.SetProtectSocketCallback(func(fd int) int {
		// Your custom socket protection logic here
		// For example, on Android you might call VpnService.protect(fd)
		// On iOS you might bind the socket to a specific interface
		// On Linux you might use SO_BINDTODEVICE
		// On Windows you might use Wintun
		log.Printf("Protecting socket fd: %d", fd)
		return 0 // Return 0 on success, non-zero on failure
	})

	// TrustTunnel TOML configuration
	// This is a minimal example - adjust according to your needs
	config := `
[server]
address = "your-server.com"
port = 443

[credentials]
username = "your-username"
password = "your-password"

[listener]
type = "socks"
address = "127.0.0.1:1080"

[upstream]
protocol = "http2"
location = "us-east"
`

	// Start the VPN
	log.Println("Starting VPN...")
	if err := bridge.Start(config); err != nil {
		log.Fatalf("Failed to start VPN: %v", err)
	}

	// Wait for a while
	log.Println("VPN is running. Press Ctrl+C to stop.")
	time.Sleep(30 * time.Second)

	// Stop the VPN
	log.Println("Stopping VPN...")
	if err := bridge.Stop(); err != nil {
		log.Fatalf("Failed to stop VPN: %v", err)
	}

	log.Println("VPN stopped.")
}
